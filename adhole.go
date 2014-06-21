package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
)

func runServerTCP(proto string, addr string, port int) {
	fulladdr := fmt.Sprintf("%s:%d", addr, port)

	srv, err := net.Listen(proto, fulladdr)
	if err != nil {
		log.Fatal(err)
	}
	defer srv.Close()

	log.Printf("Started %s server at %s\n", proto, fulladdr)
	for {
		conn, err := srv.Accept()
		if err != nil {
			log.Println("ERROR:", proto, err)
			continue
		}

		log.Println("New connection", proto, conn)
		go handleTCP(conn)
	}

	panic("not reachable")
}

func runServerUDP(proto string, addr string, port int) {
	fulladdr := fmt.Sprintf("%s:%d", addr, port)
	udpaddr := &net.UDPAddr{
		IP:   net.ParseIP(addr),
		Port: port,
	}

	srv, err := net.ListenUDP(proto, udpaddr)
	if err != nil {
		log.Fatal(err)
	}
	defer srv.Close()

	log.Printf("Started %s server at %s\n", proto, fulladdr)
	buf := make([]byte, 64)
	oobuf := make([]byte, 64)
	for {
		_, _, _, addr, err := srv.ReadMsgUDP(buf, oobuf)
		if err != nil {
			log.Println("ERROR:", proto, err)
			continue
		}

		log.Println("New message", proto, addr)
		go handleUDP(buf, addr)
	}

	panic("not reachable")
}

func handleUDP(payload []byte, addr *net.UDPAddr) {
	dump(payload)
}

func handleTCP(c net.Conn) {
	buf := make([]byte, 64)
	_, err := io.ReadAtLeast(c, buf, 12)
	if err != nil {
		log.Println("ERROR:", err)
		return
	}

	dump(buf)
	log.Println("Connection closed tcp", c)
	c.Close()
}

func dump(data []byte) {
	dumper := hex.Dumper(os.Stdout)
	io.Copy(dumper, bytes.NewReader(data))
	dumper.Close()
}

func main() {

	go runServerTCP("tcp", "192.168.0.11", 5353)
	go runServerUDP("udp", "192.168.0.11", 5354)

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
