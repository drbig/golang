package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
)

type Header struct {
	Id      uint16
	Flags   uint16
	Qtcount uint16
	Ancount uint16
	Arcount uint16
	Adcount uint16
}

type Question struct {
	Query  string
	Qtype  uint16
	Qclass uint16
}

func (h *Header) String() string {
	return fmt.Sprintf("Id: %X Flags: %b Counts: %d/%d/%d/%d",
		h.Id, h.Flags, h.Qtcount, h.Ancount, h.Arcount, h.Adcount)
}

func parseHeader(raw []byte) (hdr *Header, err error) {
	if len(raw) < 12 {
		err = errors.New(fmt.Sprintf("parseHeader: not enough raw data"))
		return
	}

	hdr = &Header{}
	reader := bytes.NewReader(raw[:12])
	err = binary.Read(reader, binary.BigEndian, hdr)

	return
}

func runServerDNS(addr string, port int) {
	fulladdr := fmt.Sprintf("%s:%d", addr, port)
	udpaddr := &net.UDPAddr{
		IP:   net.ParseIP(addr),
		Port: port,
	}

	srv, err := net.ListenUDP("udp", udpaddr)
	if err != nil {
		log.Fatal(err)
	}
	defer srv.Close()

	log.Println("Started DNS server at", fulladdr)
	buf := make([]byte, 64)
	oobuf := make([]byte, 64)
	for {
		_, _, _, addr, err := srv.ReadMsgUDP(buf, oobuf)
		if err != nil {
			log.Println("DNS ERROR:", err)
			continue
		}

		log.Println("DNS: New message from", addr)
		go handleDNS(buf, addr)
	}

	panic("not reachable")
}

func handleDNS(payload []byte, addr *net.UDPAddr) {
	header, err := parseHeader(payload)
	if err != nil {
		log.Println("DNS ERROR:", err)
	} else {
		fmt.Println("Header:", header)
	}

	dump(payload)
}

func dump(data []byte) {
	dumper := hex.Dumper(os.Stdout)
	io.Copy(dumper, bytes.NewReader(data))
	dumper.Close()
}

func main() {
	go runServerDNS("192.168.0.11", 5354)

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)

forever:
	for {
		select {
		case <-sig:
			log.Println("Signal received, stopping")
			break forever
		}
	}
}
