//
// Getlinks
//

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
)

var (
	verbose      bool
	follow       bool
	follow_rxp   *regexp.Regexp
	follow_limit int
	uri          *url.URL
	rxps         []*regexp.Regexp
)

func die(msg string, code int) {
	log.Fatalln(msg)
	os.Exit(code)
}

func make_regexp(arg string) *regexp.Regexp {
	rxp, err := regexp.Compile(arg)
	if err != nil {
		die("Error compiling regexp '"+arg+"'!", 2)
	}
	if rxp.NumSubexp() != 1 {
		die("Regexp '"+arg+"' has to have exactly one group!", 2)
	}
	return rxp
}

func setup() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] url regexp regexp...\n", os.Args[0])
		flag.PrintDefaults()
	}

	var follow_str string
	flag.BoolVar(&verbose, "v", false, "Be verbose on stderr")
	flag.StringVar(&follow_str, "f", "", "Follow link regexp")
	flag.IntVar(&follow_limit, "l", 0, "Limit following to n times (0 = no limit)")
	flag.Parse()

	if follow_str != "" {
		follow_rxp = make_regexp(follow_str)
		follow = true
	} else {
		follow = false
	}

	args := flag.Args()
	n := len(args)
	if n < 2 {
		die("You have to specify at least url and one regexp!", 5)
	}

	var err error
	uri_str := args[0]
	uri, err = url.Parse(uri_str)
	if err != nil {
		die("Error parsing primary url '"+uri_str+"'!", 5)
	}

	rxps = make([]*regexp.Regexp, n-1)
	for i := 1; i < n; i++ {
		rxps[i-1] = make_regexp(args[i])
	}
}

func say(what string) {
	if verbose {
		log.Println(what)
	}
}

func fetch(uri *url.URL) (string, error) {
	res, err := http.Get(uri.String())
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return bytes.NewBuffer(body).String(), nil
}

func process_page(base *url.URL, page int, body string, rxp *regexp.Regexp, ctrl chan bool) {
	say(fmt.Sprintf("Page %d - processing regexp '%s'", page, rxp.String()))
	matches := rxp.FindAllStringSubmatch(body, -1)
	say(fmt.Sprintf("Page %d - found %d matches", page, len(matches)))
	for i := 0; i < len(matches); i++ {
		uri_str := matches[i][1]
		this_uri, err := url.Parse(uri_str)
		if err != nil {
			log.Fatalln("Error parsing url '" + uri_str + "'!")
			continue
		}
		if !this_uri.IsAbs() {
			this_uri = base.ResolveReference(this_uri)
		}
		fmt.Println(this_uri.String())
	}
	ctrl <- true
}

func main() {
	setup()

	body, err := fetch(uri)
	if err != nil {
		die("Error loading the primary url!", 5)
	}

	workers := 0
	pages := 1
	ctrl := make(chan bool)

	for {
		for _, rxp := range rxps {
			go process_page(uri, pages, body, rxp, ctrl)
			workers += 1
		}

		if follow {
			if follow_limit > 0 && pages == follow_limit {
				say(fmt.Sprintf("Finished processing %d pages", pages))
				break
			}
			next := follow_rxp.FindStringSubmatch(body)
			if next != nil {
				next_uri, err := url.Parse(next[1])
				if err != nil {
					log.Fatalln("Error parsing url '" + next[1] + "'!")
					break
				}
				if !next_uri.IsAbs() {
					uri = uri.ResolveReference(next_uri)
				} else {
					uri = next_uri
				}
				body, err = fetch(uri)
				if err != nil {
					log.Fatalln("Error fetching url '" + uri.String() + "'!")
					break
				}
				pages++
			} else {
				say(fmt.Sprintf("No more next pages found at page %d", pages))
				break
			}
		} else {
			break
		}
	}

	for _ = range ctrl {
		workers--
		if workers == 0 {
			break
		}
	}
}
