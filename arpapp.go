// Arpapp
// dRbiG
// See LICENSE.txt

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"time"
)

type ArpEntry struct {
	online bool
	stamp  time.Time
}

const (
	ARPREGEX  = ".*? \\((.*?)\\) "
	LOGSIZE   = 16
	HTMLSTART = `<html><head><title>arpapp</title>
<style>body{background:black;color:#d0d0d0}span.online{color:green}span.offline{color:red}</style>
</head><body><pre><b>arpapp</b>

`
	HTMLEND = `</pre></body></html>`
)

var (
	host     string
	port     int
	interval time.Duration
	decay    int
	arplog   map[string]ArpEntry
	arpregex *regexp.Regexp
	decayrex *regexp.Regexp
)

func die(msg string, code int) {
	log.Fatalln(msg)
	os.Exit(code)
}

func setup() {
	var err error
	var intervalstr string
	var decayrexstr string

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] PORT\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.StringVar(&host, "h", "127.0.0.1", "HTTP server bind HOST")
	flag.StringVar(&intervalstr, "i", "10m", "Scan INTERVAL")
	flag.IntVar(&decay, "d", 72, "Forget after DECAY hours")
	flag.StringVar(&decayrexstr, "r", "", "Decay only IPs matching REGEXP")
	flag.Parse()

	if len(flag.Args()) < 1 {
		die("You have to specify a port!", 2)
	}

	interval, err = time.ParseDuration(intervalstr)
	if err != nil {
		die("Couldn't parse interval!", 2)
	}

	port, err = strconv.Atoi(flag.Args()[0])
	if err != nil || port <= 0 {
		die("Port has to be a positive integer!", 2)
	}

	if (len(decayrexstr) > 0) && (decay > 0) {
		decayrex, err = regexp.Compile(decayrexstr)
		if err != nil {
			die("Failed to compile regexp '"+decayrexstr+"'!", 2)
		}
	} else {
		decayrex = nil
	}

	arpregex = regexp.MustCompile(ARPREGEX)
	arplog = make(map[string]ArpEntry, LOGSIZE)
}

func scan() {
	data, err := exec.Command("arp", "-a").Output()
	if err != nil {
		log.Println("ERROR: running 'arp -a'")
		return
	}

	matches := arpregex.FindAllStringSubmatch(string(data), -1)
	seen := make(map[string]bool, len(matches))
	for i := 0; i < len(matches); i++ {
		ip := matches[i][1]
		if exec.Command("ping", "-c1", "-W1", ip).Run() == nil {
			seen[ip] = true
		} else {
			continue
		}
	}
	for ip, _ := range seen {
		if _, present := arplog[ip]; present {
			if !arplog[ip].online {
				arplog[ip] = ArpEntry{online: true, stamp: time.Now()}
			}
		} else {
			arplog[ip] = ArpEntry{online: true, stamp: time.Now()}
		}
	}
	for ip, entry := range arplog {
		if _, present := seen[ip]; !present {
			if entry.online {
				arplog[ip] = ArpEntry{online: false, stamp: time.Now()}
			}
		}
		if (decayrex != nil) && !entry.online {
			if decayrex.MatchString(ip) {
				duration := time.Since(entry.stamp)
				if duration.Hours() >= float64(decay) {
					delete(arplog, ip)
				}
			}
		}
	}
}

func render(out http.ResponseWriter, req *http.Request) {
	keys := make([]string, len(arplog))
	i := 0
	for ip, _ := range arplog {
		keys[i] = ip
		i++
	}
	sort.Strings(keys)

	fmt.Fprintf(out, HTMLSTART)
	for _, ip := range keys {
		entry := arplog[ip]
		duration := time.Since(entry.stamp)

		fmt.Fprintf(out, "<span class='")
		if entry.online {
			fmt.Fprintf(out, "online")
		} else {
			fmt.Fprintf(out, "offline")
		}
		fmt.Fprintf(out, "'>%-15s</span> since %s (%s)\n", ip, entry.stamp.Format("2006-01-02 15:04:05 MST"), duration.String())
	}
	fmt.Fprintf(out, "\ninterval: %s", interval)
	fmt.Fprintf(out, HTMLEND)
}

func main() {
	setup()

	http.HandleFunc("/", render)
	go http.ListenAndServe(fmt.Sprintf("%s:%d", host, port), nil)

	for {
		scan()
		time.Sleep(interval)
	}
}
