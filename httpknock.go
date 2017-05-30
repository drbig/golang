// HttpKnock
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
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	//	OPEN_FW_CMD_FMT  = `sudo ipfw table 1 add %s`
	//	CLOSE_FW_CMD_FMT = `sudo ipfw table 1 delete %s`
	OPEN_FW_CMD_FMT  = `echo open %s`
	CLOSE_FW_CMD_FMT = `echo close %s`
	KEEP_DURATION    = time.Duration(5) * time.Second
	HOST             = `0.0.0.0`
	PORT             = 9996
	PASSWORD_VAR     = `HK_PASSWORD`
	PASSWORD_KEY     = `key`
)

const (
	VERSION  = `0.1`
	HELP_FMT = `Usage: %s (options)
httpknock v%s, see LICENSE.txt

Set password using %s env variable.

`
)

type timerMap struct {
	mu sync.Mutex
	ts map[string]*time.Timer
}

var (
	flagKeepDuration time.Duration
	flagHost         string
	flagPort         int
)

var (
	password string
	timers   timerMap
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, HELP_FMT, os.Args[0], VERSION, PASSWORD_VAR)
		flag.PrintDefaults()
	}

	flag.DurationVar(&flagKeepDuration, "kd", KEEP_DURATION, "Keep open for given duration.")
	flag.StringVar(&flagHost, "h", HOST, "Host to bind to.")
	flag.IntVar(&flagPort, "p", PORT, "Port to bind to.")

	timers.ts = make(map[string]*time.Timer, 256)
}

func main() {
	var ok bool

	if password, ok = get_password(); !ok {
		fmt.Fprintf(os.Stderr, "Variable %s not set. Can't run without password.\n", PASSWORD_VAR)
		os.Exit(1)
	}

	flag.Parse()
	go runHTTPServer()
	sigwait()

	log.Println("HttpKnock stopped.")
}

func handleOpen(w http.ResponseWriter, req *http.Request) {
	if !auth_request(w, req) {
		return
	}

	ip := get_ip(req.RemoteAddr)
	if !run_fw_cmd(OPEN_FW_CMD_FMT, ip) {
		fmt.Fprintln(w, "FAILED")
		return
	}

	duration := flagKeepDuration
	if val := req.FormValue("for"); val != "" {
		user_duration, err := time.ParseDuration(val)
		if err != nil {
			log.Printf("Failed to parse duration: %s", err)
			fmt.Fprintln(w, "Failed to parse duration, using default.")
		} else {
			duration = user_duration
		}
	}

	timers.mu.Lock()
	timers.ts[ip] = time.AfterFunc(duration, func() {
		log.Printf("Closing FW for %s after %s timeout...", ip, duration)
		run_fw_cmd(CLOSE_FW_CMD_FMT, ip)
		timers.mu.Lock()
		delete(timers.ts, ip)
		timers.mu.Unlock()
	})
	timers.mu.Unlock()

	until := time.Now().Add(duration)
	log.Printf("Added %s until %s\n", ip, until)

	fmt.Fprintf(w, "Added %s until %s.\n", ip, until)
	fmt.Fprintln(w, "OK")
}

func handleClose(w http.ResponseWriter, req *http.Request) {
	if !auth_request(w, req) {
		return
	}

	var ip string
	if val := req.FormValue("ip"); val != "" {
		ip = val
	} else {
		ip = get_ip(req.RemoteAddr)
	}

	timers.mu.Lock()
	if _, ok := timers.ts[ip]; ok {
		if !run_fw_cmd(CLOSE_FW_CMD_FMT, ip) {
			fmt.Fprintln(w, "FAILED")
			timers.mu.Unlock()
			return
		}

		timers.ts[ip].Stop()
		delete(timers.ts, ip)

		log.Printf("Client %s killed timer for: %s\n", req.RemoteAddr, ip)
		fmt.Fprintln(w, "OK")
	} else {
		log.Printf("Client %s tried to unblock non-blocked ip: %s\n", req.RemoteAddr, ip)
		fmt.Fprintf(w, "IP %s is not open.\n", ip)
		fmt.Fprintln(w, "FAILED")
	}

	timers.mu.Unlock()
}

func run_fw_cmd(cmd_fmt string, addr string) bool {
	cmd_str := fmt.Sprintf(cmd_fmt, addr)
	cmd_args := strings.Split(cmd_str, " ")

	if output, err := exec.Command(cmd_args[0], cmd_args[1:]...).Output(); err != nil {
		log.Printf("Command '%s' failed: %s\nOutput: %s\n", cmd_str, err, string(output))
		return false
	}

	log.Printf("Command '%s' succeeded", cmd_str)
	return true
}

func runHTTPServer() {
	addr := fmt.Sprintf("%s:%d", flagHost, flagPort)
	http.HandleFunc("/open", handleOpen)
	http.HandleFunc("/close", handleClose)
	log.Println("Starting HTTP server at", addr)
	log.Fatalln(http.ListenAndServe(addr, nil))
}

func auth_request(w http.ResponseWriter, req *http.Request) bool {
	if val := req.FormValue(PASSWORD_KEY); val == password {
		log.Printf("Authorized access to %s form %s\n", req.RequestURI, req.RemoteAddr)
		return true
	}

	log.Printf("Unauthorized access to %s from %s\n", req.RequestURI, req.RemoteAddr)
	http.NotFound(w, req)
	return false
}

func get_password() (string, bool) {
	for _, e := range os.Environ() {
		kv := strings.Split(e, "=")
		if kv[0] == PASSWORD_VAR {
			return kv[1], true
		}
	}

	return "", false
}

func get_ip(addr string) string {
	return strings.Split(addr, ":")[0]
}

func sigwait() {
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	stop_sig := <-sig
	log.Printf("Received signal: %s\n", stop_sig)
}
