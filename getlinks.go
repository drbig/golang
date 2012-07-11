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
  "strings"
)

var (
  verbose bool
  follow_arg string
  follow_type string
  follow_types = []string{"text", "url", "class", "id"}
  tasks map[string]*regexp.Regexp
)


func die(msg string, code int) {
  log.Fatalln(msg)
  os.Exit(code)
}

func setup() {
  flag.BoolVar(&verbose, "v", false, "Be verbose")
  flag.StringVar(&follow_arg, "f", "", "Follow link regexp")
  flag.StringVar(&follow_type, "t", "text", "Follow link by " + strings.Join(follow_types, "|"))
  flag.Parse()
  args := flag.Args()
  if len(args) == 0 || len(args) % 2 != 0 {
    die("You have to specify at least one url regexp pair!", 5)
  } else {
    n := len(args)
    tasks = make(map[string]*regexp.Regexp, n / 2)
    for i := 0; i < n; i += 2 {
      arg := args[i+1]
      rxp, err := regexp.Compile(arg)
      if err != nil {
        die("Error parsing regexp '" + arg + "'!", 2)
      } else {
        tasks[args[i]] = rxp
      }
    }
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
  if follow_arg != "" {
    say("Following type: " + follow_type)
    say("Following regexp: " + follow_arg)
  }
  for url, rxp := range tasks {
    say("Processing link '" + url + "' with regexp '" + rxp.String() + "'")
    body, err := fetch(url)
    if err != nil {
      log.Fatalln("Error fetching url '" + url + "'!")
    }
    //fmt.Println(body)
    matches := rxp.FindAllStringSubmatch(body, -1)
    fmt.Printf("Found %d\n", len(matches))
    for i := 0; i < len(matches); i++ {
      fmt.Println(matches[i][1])
    }
  }
}
