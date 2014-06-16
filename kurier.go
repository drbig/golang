// What's the delivery status?
// dRbiG, 2014
// See LICENSE.txt

package main

import (
	"errors"
	"fmt"
	"github.com/kennygrant/sanitize"
	"github.com/moovweb/gokogiri"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
)

type Kurier interface {
	IsMatch(string) bool
	Check(string) (string, error)
}

type Service struct {
	Name      string
	Matcher   string
	URL       string
	XPath     string
	Extractor *regexp.Regexp
}

func (s *Service) IsMatch(id string) bool {
	matched, err := regexp.MatchString(s.Matcher, id)
	if err != nil {
		panic(err)
	}
	return matched
}

func (s *Service) Check(id string) (status string, err error) {
	body, err := downloadPage(fmt.Sprintf(s.URL, id))
	if err != nil {
		return
	}

	if s.Extractor != nil {
		parts := s.Extractor.FindSubmatch(body)
		if parts == nil {
			return "", nil
		}

		status = string(parts[1])
	} else {
		doc, err := gokogiri.ParseHtml(body)
		if err != nil {
			return "", err
		}
		defer doc.Free()

		res, err := doc.Search(s.XPath)
		if err != nil {
			return "", err
		}
		if len(res) < 1 {
			return "", nil
		}

		status = sanitize.HTML(res[0].String())
		status = replacer.ReplaceAllString(status, " ")
		status = strings.TrimSpace(status)
	}

	return
}

func downloadPage(url string) (body []byte, err error) {
	res, err := client.Get(url)
	if err != nil {
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		err = errors.New(fmt.Sprintf("DownloadPage: %d %s", res.StatusCode, url))
		return
	}

	body, err = ioutil.ReadAll(res.Body)
	return
}

func checkService(id string, s Service) {
	res, err := s.Check(id)
	if (err == nil) && (res != "") {
		fmt.Printf("%20s %-8s %s\n", id, s.Name, res)
	}
	wg.Done()
}

var (
	client   *http.Client
	replacer *regexp.Regexp
	wg       sync.WaitGroup
)

func main() {
	services := [...]Service{
		Service{
			"DHL",
			"^\\d{11}$",
			"http://www.dhl.com.pl/sledzenieprzesylkikrajowej/szukaj.aspx?m=0&sn=%s",
			"//*[@id='middle']/table/tbody/tr[2]/td[4]/text()[1]",
			nil,
		},
		Service{
			"DPD",
			"^\\w{14}$",
			"http://www.dpd.com.pl/tracking.asp?p1=%s&przycisk=Wyszukaj",
			"//table[@class='subpage_modules']/tr[2]/td[3]",
			nil,
		},
		Service{
			"SIÓDEMKA",
			"^\\d{13}$",
			"https://siodemka.com/tracking/%s/",
			"//*[@id='page']/div[2]/table[2]/tbody/tr[4]/td[4]",
			nil,
		},
		Service{
			"UPS",
			"^1Z\\w{16}$",
			"http://wwwapps.ups.com/WebTracking/track?loc=pl_PL&HTMLVersion=5.0&Requester=UPSHome&WBPM_lid=homepage/ct1.html_pnl_trk&trackNums=%s&track.x=Monitoruj",
			"//*[@id='tt_spStatus']/text()",
			nil,
		},
		Service{
			"GLS",
			"^\\d{11}$",
			"https://gls-group.eu/app/service/open/rest/PL/pl/rstt001?match=%s&caller=witt002",
			"",
			regexp.MustCompile("\"statusText\":\"(.*?)\""),
		},
	}

	client = &http.Client{}
	replacer = regexp.MustCompile("\\s{2,}|\\t+|\\\\n")

	for _, id := range os.Args[1:] {
		for _, s := range services {
			if s.IsMatch(id) {
				wg.Add(1)
				go checkService(id, s)
			}
		}
	}

	wg.Wait()
}
