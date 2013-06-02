//
// Dumping HTTP Proxy
//

package main

import (
	"flag"
	"fmt"
	"github.com/elazarl/goproxy"
	"log"
	"net/http"
	"os"
	"strconv"
)

func die(msg string, code int) {
	log.Fatalln(msg)
	os.Exit(code)
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] PORT\n", os.Args[0])
		flag.PrintDefaults()
	}

	var host string
	flag.StringVar(&host, "h", "127.0.0.1", "HTTP server bind HOST")
	flag.Parse()
	if len(flag.Args()) < 1 {
		die("You have to specify a port!", 2)
	}
	port, err := strconv.Atoi(flag.Args()[0])
	if err != nil || port <= 0 {
		die("Port has to be a positive integer!", 2)
	}

	proxy := goproxy.NewProxyHttpServer()
	proxy.OnRequest().DoFunc(
		func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			fmt.Println(req.RequestURI)
			return req, nil
		})
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", host, port), proxy))
}
