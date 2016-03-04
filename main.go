package main

import (
	"bufio"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"os"
	"strings"
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

type smtpTx struct {
	mailFrom string
	rcptTo   []string
	data     []string
}

func handleConnection(conn net.Conn) {
	conns.Lock()
	conns.i++
	id := conns.i
	conns.Unlock()

	log.Printf("Handling %v", id)
	defer conn.Close()
	defer log.Printf("Closing %v", id)

	r := textproto.NewReader(bufio.NewReader(conn))
	w := textproto.NewWriter(bufio.NewWriter(conn))

	if err := w.PrintfLine("220 foo"); err != nil {
		log.Fatal(err)
	}

	var s smtpTx
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
			return
		}
		log.Print(line)
		switch split := strings.Split(line, ":"); split[0] {
		case "MAIL FROM":
			s.mailFrom = split[1]
		case "RCPT TO":
			s.rcptTo = append(s.rcptTo, split[1])
		case "DATA":
			if err := w.PrintfLine("354 foo"); err != nil {
				log.Print(err)
				return
			}
			lines, err := r.ReadDotLines()
			if err != nil {
				log.Print(err)
				return
			}
			log.Printf("%#v", lines)
			s.data = lines
		case "QUIT":
			if err := w.PrintfLine("221 foo"); err != nil {
				log.Print(err)
			}
			go sendMail(s)
			return
		}

		if err := w.PrintfLine("250 foo"); err != nil {
			log.Print(err)
			return
		}
	}
}

var (
	smtpUsername = os.Getenv("SMTP_USERNAME")
	smtpPassword = os.Getenv("SMTP_PASSWORD")
	smtpServer   = os.Getenv("SMTP_SERVER")
	smtpPort     = os.Getenv("SMTP_PORT")
)

func sendMail(s smtpTx) {
	auth := smtp.PlainAuth("", smtpUsername, smtpPassword, smtpServer)
	from := s.mailFrom

	to := []string{"yannsalaun1@gmail.com"}

	var plus string
	for _, v := range s.rcptTo {
		v = strings.Trim(v, "<>")
		if strings.HasSuffix(v, "@yannsalaun.com") {
			plus = v[:strings.Index(v, "@")]
			break
		}
	}

	var hdrs string
	msg, err := mail.ReadMessage(strings.NewReader(strings.Join(s.data, "\r\n")))
	if err != nil {
		log.Print(err)
	}
	for k, v := range msg.Header {
		if strings.HasPrefix(k, "X-") || k == "Dkim-Signature" ||
			k == "Message-Id" || k == "Received" {
			continue
		}
		if k == "To" {
			v[0] = "yannsalaun1+" + plus + "@gmail.com"
		}
		hdrs += k + ": " + v[0] + "\r\n"
	}

	bytes, err := ioutil.ReadAll(io.MultiReader(strings.NewReader(hdrs), msg.Body))
	if err != nil {
		log.Print(err)
	}
	if err := smtp.SendMail(smtpServer+":"+smtpPort, auth, from, to, bytes); err != nil {
		log.Print(err)
	} else {
		log.Print("mail sent")
	}
}
