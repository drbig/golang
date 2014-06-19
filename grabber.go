// Declarative web grabber
// dRbiG, 2014
// See LICENSE.txt

package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/moovweb/gokogiri"
	"github.com/moovweb/gokogiri/html"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Action struct {
	XPath  string
	Mode   string
	Action string
	Do     *Action
}

type Target struct {
	Name string
	URL  string
	Bail int
	Path string
	Do   *Action
}

type Stats struct {
	Number int
	Size   int64
	Took   time.Duration
	Mtx    sync.Mutex
}

func (s *Stats) Update(n int64, took time.Duration) {
	s.Mtx.Lock()
	s.Number++
	s.Size += n
	s.Took += took
	s.Mtx.Unlock()
}

func (s *Stats) IsEmpty() (b bool) {
	s.Mtx.Lock()
	if s.Number < 1 {
		b = true
	}
	s.Mtx.Unlock()
	return
}

func (s *Stats) String() string {
	s.Mtx.Lock()
	num := s.Number
	size := float64(s.Size) / (1024.0 * 1024.0)
	speed := size / s.Took.Seconds()
	s.Mtx.Unlock()
	return fmt.Sprintf("%d files for %0.2f MB with avg. dl. speed %0.3f MB/s", num, size, speed)
}

var (
	downloader chan *url.URL
	root       string
	bail       int
	counter    int
	stats      *Stats
	client     *http.Client
	mtx        sync.RWMutex
	wg         sync.WaitGroup
)

func LoadRules(name string) (t []Target, err error) {
	handle, err := os.Open(name)
	if err != nil {
		return
	}
	defer handle.Close()

	raw, err := ioutil.ReadAll(handle)
	if err != nil {
		return
	}

	err = json.Unmarshal(raw, &t)
	if err != nil {
		return
	}

	return
}

func DownloadPage(path string) (doc *html.HtmlDocument, err error) {
	res, err := client.Get(path)
	if err != nil {
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		err = errors.New(fmt.Sprintf("DownloadPage: %d %s", res.StatusCode, path))
		return
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	doc, err = gokogiri.ParseHtml(data)
	return
}

func DownloadFile(target string, fullpath string) (n int64, took time.Duration, err error) {
	start := time.Now()

	res, err := client.Get(target)
	if err != nil {
		return
	}
	defer res.Body.Close()

	handle, err := os.Create(fullpath)
	if err != nil {
		return
	}
	defer handle.Close()

	n, err = io.Copy(handle, res.Body)
	took = time.Since(start)
	return
}

func DoAction(base *url.URL, target string, act *Action) (res *url.URL, err error) {
	res = &url.URL{}

	if act.Action == "raw" {
		fmt.Println(target)
		return
	}

	urlTarget, err := url.Parse(target)
	if err != nil {
		return
	}
	if len(urlTarget.Path) < 1 {
		return
	}

	if urlTarget.IsAbs() {
		*res = *urlTarget
	} else {
		*res = *base
		if urlTarget.Path[0] == '/' {
			res.Path = urlTarget.Path
		} else {
			res.Path += urlTarget.Path
		}
	}

	switch act.Action {
	case "print":
		fmt.Println(res)
	case "log":
		log.Println(res)
	case "download":
		if bail > 0 {
			mtx.RLock()
			cnt := counter
			mtx.RUnlock()
			if cnt >= bail {
				err = errors.New(fmt.Sprintf("DoAction: bailout after %d", counter))
				return
			}
		}
		downloader <- res
	case "none":
		// nop
	default:
		err = errors.New(fmt.Sprintf("DoAction: unknown action %s", act.Action))
	}

	return
}

func Process(base *url.URL, act *Action) (err error) {
	doc, err := DownloadPage(base.String())
	if err != nil {
		return err
	}
	defer doc.Free()

	res, err := doc.Search(act.XPath)
	if err != nil {
		return err
	}

	switch act.Mode {
	case "follow":
		if act.Do != nil {
			err = Process(base, act.Do)
			if err != nil {
				return err
			}
		}

		if len(res) < 1 {
			err = errors.New(fmt.Sprintf("Process: %s no results", base))
			return err
		}

		target, err := DoAction(base, res[0].String(), act)
		if err != nil {
			return err
		}

		err = Process(target, act)
		if err != nil {
			return err
		}
	case "every":
		if len(res) < 1 {
			if act.Do != nil {
				err = errors.New(fmt.Sprintf("Process: %s no results", base))
				return err
			} else {
				return
			}
		}

		for _, v := range res {
			target, err := DoAction(base, v.String(), act)
			if err != nil {
				return err
			}

			if act.Do != nil {
				err = Process(target, act.Do)
				if err != nil {
					return err
				}
			}
		}
	default:
		err = errors.New(fmt.Sprintf("Process: unknown mode %s", act.Mode))
	}

	return err
}

func Downloader() {
	for target := range downloader {
		parts := strings.Split(target.Path, "/")
		name, err := url.QueryUnescape(parts[len(parts)-1])
		if err != nil {
			name = parts[len(parts)-1]
		}

		fullpath := filepath.Join(root, name)

		if _, err := os.Stat(fullpath); err == nil {
			mtx.Lock()
			counter++
			mtx.Unlock()
		} else {
			log.Println("Downloading", fullpath)
			n, took, err := DownloadFile(target.String(), fullpath)
			if err != nil {
				log.Printf("ERROR Downloader: %s\n", err)
			} else {
				stats.Update(n, took)
			}
		}
	}
	wg.Done()
}

func main() {
	start := time.Now()

	var flgDls = flag.Int("dls", 3, "number of downloaders")
	var flgLog = flag.Bool("log", false, "log to stdout")
	flag.Parse()

	if *flgLog {
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(os.Stderr)
	}

	if len(flag.Args()) != 1 {
		log.Fatalln("Please specify rules file.")
	}

	log.Println("Loading rules...")
	rs, err := LoadRules(flag.Arg(0))
	if err != nil {
		log.Fatalln(err)
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client = &http.Client{Transport: tr}

	downloader = make(chan *url.URL, *flgDls)
	stats = &Stats{}

	for i := 0; i < *flgDls; i++ {
		wg.Add(1)
		go Downloader()
	}

	log.Println("Executing...")
	for _, target := range rs {
		startTarget := time.Now()
		log.Println("Target:", target.Name)

		root, err = filepath.Abs(target.Path)
		if err != nil {
			log.Println("ERROR", err)
			continue
		}

		if _, err := os.Stat(root); os.IsNotExist(err) {
			log.Println("ERROR", err)
			continue
		}

		base, err := url.Parse(target.URL)
		if err != nil {
			log.Println("ERROR", err)
			continue
		}

		bail = target.Bail
		counter = 1

		err = Process(base, target.Do)
		if err != nil {
			log.Println("ERROR", err)
		}

		log.Printf("Finished target: %s (took %s)\n", target.Name, time.Since(startTarget))
	}

	close(downloader)
	log.Println("Waiting for downloaders to finish...")
	wg.Wait()
	if !stats.IsEmpty() {
		log.Println("Download statistics:", stats)
	}
	log.Printf("All done (took %s).", time.Since(start))
}
