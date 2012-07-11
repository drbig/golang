//
// Getlinks
//

package main

import (
  "bytes"
  "fmt"
  "flag"
  "io/ioutil"
  "log"
  "net/http"
  "os"
  "regexp"
)

var (
  verbose bool
  follow bool
  follow_rxp *regexp.Regexp
  follow_limit int
  par_limit int
  url string
  rxps []*regexp.Regexp
)


func die(msg string, code int) {
  log.Fatalln(msg)
  os.Exit(code)
}

func make_regexp(arg string) *regexp.Regexp {
  rxp, err := regexp.Compile(arg)
  if err != nil {
    die("Error compiling regexp '" + arg + "'!", 2)
  }
  if rxp.NumSubexp() != 1 {
    die("Regexp '" + arg + "' has to have exactly one group!", 2)
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
  flag.IntVar(&par_limit, "p", 5, "Limit parallel processing to n threads (0 = no limit)")
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
  url = args[0];
  rxps = make([]*regexp.Regexp, n - 1)
  for i := 1; i < n; i++ {
    rxps[i - 1] = make_regexp(args[i])
  }
}

func say(what string) {
  if verbose {
    log.Println(what)
  }
}

func fetch(url string) (string, error) {
  res, err := http.Get(url)
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

func main() {
  setup()
  say("Hello there")
  if follow {
    say("Following regexp: " + follow_rxp.String())
  }
  body, err := fetch(url)
  if err != nil {
    die("Error loading the primary url!", 5)
  }
  for _, rxp := range rxps {
    say("Processing regexp '" + rxp.String() + "'")
    matches := rxp.FindAllStringSubmatch(body, -1)
    say(fmt.Sprintf("Found %d matches", len(matches)))
    for i := 0; i < len(matches); i++ {
      fmt.Println(matches[i][1])
    }
  }
}
