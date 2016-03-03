package main

import (
	"bufio"
	"log"
	"net"
	"net/textproto"
	"sync"
)

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Listening on port 8080")
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go handleConnection(conn)
	}
}

var conns struct {
	sync.Mutex
	i int
}

func handleConnection(conn net.Conn) {
	conns.Lock()
	conns.i++
	id := conns.i
	conns.Unlock()

	log.Printf("Handling %v", id)
	defer conn.Close()

	r := textproto.NewReader(bufio.NewReader(conn))
	w := textproto.NewWriter(bufio.NewWriter(conn))

	if err := w.PrintfLine("220 foo"); err != nil {
		log.Fatal(err)
	}

	// 2016/03/03 21:38:47 EHLO localhost
	// 2016/03/03 21:38:47 MAIL FROM:<sender@example.org>
	// 2016/03/03 21:38:47 RCPT TO:<recipient@example.net>
	// 2016/03/03 21:38:47 DATA
	// 2016/03/03 21:38:47 [To: recipient@example.net Subject: discount Gophers!  This is the email body.]
	// 2016/03/03 21:38:47 QUIT
	for {
		line, err := r.ReadLine()
		if err != nil {
			log.Print(line, err)
			break
		}
		log.Print(line)

		if line == "DATA" {
			if err := w.PrintfLine("354 foo"); err != nil {
				log.Print(err)
				break
			}
			lines, err := r.ReadDotLines()
			if err != nil {
				log.Print(err)
				break
			}
			log.Printf("%+v", lines)
		} else if line == "QUIT" {
			if err := w.PrintfLine("221 foo"); err != nil {
				log.Print(err)
			}
			break
		}

		if err := w.PrintfLine("250 foo"); err != nil {
			log.Print(err)
			break
		}

	}

	log.Printf("Closing %v", id)
}
