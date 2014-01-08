//
// WakeOnLan in Go.
// 2014-01-07
// dRbiG
//

package main

import (
	"fmt"
	"net"
	"os"
)

const (
	BRDIP = "192.168.0.255"
)

var (
	mactab = map[string]string{
		"bebop.l": "70:5A:B6:94:8F:59",
		"gamer.l": "00:25:22:f4:0b:0e",
		"lore.l":  "00:40:ca:6d:16:07",
		"nox.l":   "00:40:63:D5:8B:65",
		"rpi.l":   "B8:27:EB:0D:EB:01",
	}
	payload []byte
)

func makepayload(mac string) bool {
	var x byte

	if len(mac) != 17 {
		return false
	}

	for i := 0; i < 6; i++ {
		_, err := fmt.Sscanf(mac[3*i:3*i+2], "%2X", &x)
		if err != nil {
			return false
		}
		for c := 0; c < 16; c++ {
			payload[6*c+6+i] = x
		}
	}

	return true
}

func main() {
	var addr string

	payload = make([]byte, 102)
	for i := 0; i < 6; i++ {
		payload[i] = 255
	}

	target, _ := net.ResolveUDPAddr("udp", BRDIP+":9")
	sock, _ := net.DialUDP("udp", nil, target)

	for i := 1; i < len(os.Args); i++ {
		if val, present := mactab[os.Args[i]]; present {
			addr = val
		} else {
			addr = os.Args[i]
		}
		if !makepayload(addr) {
			fmt.Printf("%s: parse error!\n", os.Args[i])
		} else {
			for x := 0; x < 3; x++ {
				sock.Write(payload)
			}
		}
	}
}
