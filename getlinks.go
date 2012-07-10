//
// Getlinks
//

package main

import (
  "flag"
  "log"
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
    n := len(args) / 2
    tasks = make(map[string]*regexp.Regexp, n)
    for i := 0; i < n; i += 1 {
      arg := args[i+1]
      regexp, err := regexp.Compile(arg)
      if err != nil {
        die("Error parsing regexp '" + arg + "'!", 2)
      } else {
        tasks[args[i]] = regexp
      }
    }
  }
}

func say(what string) {
  if verbose {
    log.Println(what)
  }
}

func main() {
  setup()
  say("Hello there")
  if follow_arg != "" {
    say("Following type: " + follow_type)
    say("Following regexp: " + follow_arg)
  }
  for url, regexp := range tasks {
    say("Processing link '" + url + "' with regexp '" + regexp.String() + "'")
  }
}
