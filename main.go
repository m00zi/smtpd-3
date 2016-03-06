package main

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"os"
	"strings"
)

func main() {
	ln, err := net.Listen("tcp", ":2525")
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Listening on port 2525")
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go handle(conn)
	}
}

type smtpSession struct {
	mailFrom string
	rcptTo   []string
	data     []string
}

func handle(conn net.Conn) {
	defer conn.Close()
	log.Printf("Handling %+v", conn.RemoteAddr())
	defer log.Printf("Closing %+v", conn.RemoteAddr())

	buf := &bytes.Buffer{}
	r := textproto.NewReader(bufio.NewReader(io.TeeReader(conn, buf)))
	w := textproto.NewWriter(bufio.NewWriter(io.MultiWriter(conn, buf)))
	defer io.Copy(os.Stdout, buf)

	session, err := smtpHandle(w, r)
	if err != nil {
		log.Print(err)
		return
	}
	go sendMail(session)
}

func smtpHandle(w *textproto.Writer, r *textproto.Reader) (*smtpSession, error) {
	s := &smtpSession{}
	if err := w.PrintfLine("220 foo"); err != nil {
		return s, err
	}

	for {
		line, err := r.ReadLine()
		if err != nil {
			return s, err
		}
		switch split := strings.Split(line, ":"); split[0] {
		case "MAIL FROM":
			s.mailFrom = split[1]
		case "RCPT TO":
			s.rcptTo = append(s.rcptTo, split[1])
		case "DATA":
			if err := w.PrintfLine("354 foo"); err != nil {
				return s, err
			}
			lines, err := r.ReadDotLines()
			if err != nil {
				return s, err
			}
			s.data = lines
		case "QUIT":
			err := w.PrintfLine("221 foo")
			return s, err
		}

		if err := w.PrintfLine("250 foo"); err != nil {
			return s, err
		}
	}
}

var (
	smtpUsername = os.Getenv("SMTP_USERNAME")
	smtpPassword = os.Getenv("SMTP_PASSWORD")
	smtpServer   = os.Getenv("SMTP_SERVER")
	smtpPort     = os.Getenv("SMTP_PORT")

	fakeRcptDomain = os.Getenv("FAKE_RCPT_DOMAIN")
	trueRcptLocal  = os.Getenv("TRUE_RCPT_LOCAL")
	trueRcptDomain = os.Getenv("TRUE_RCPT_DOMAIN")
)

func init() {
	if smtpUsername == "" {
		log.Print("SMTP_USERNAME not set")
	}
	if smtpPassword == "" {
		log.Print("SMTP_PASSWORD not set")
	}
	if smtpServer == "" {
		log.Print("SMTP_SERVER not set")
	}
	if smtpPort == "" {
		log.Print("SMTP_PORT not set")
	}
	if fakeRcptDomain == "" {
		log.Print("FAKE_RCPT_DOMAIN not set")
	}
	if trueRcptLocal == "" {
		log.Print("TRUE_RCPT_LOCAL not set")
	}
	if trueRcptDomain == "" {
		log.Print("TRUE_RCPT_DOMAIN not set")
	}
}

func sendMail(s *smtpSession) {
	auth := smtp.PlainAuth("", smtpUsername, smtpPassword, smtpServer)
	from := s.mailFrom

	msg, err := mail.ReadMessage(strings.NewReader(strings.Join(s.data, "\r\n")))
	if err != nil {
		log.Print(err)
	}

	// Remove headers that block sending
	for k := range msg.Header {
		if strings.HasPrefix(k, "X-") || k == "Dkim-Signature" ||
			k == "Message-Id" || k == "Received" {
			delete(msg.Header, k)
		}
	}

	// Compute recipient address
	var rcptAddr string
	for _, v := range s.rcptTo {
		v = strings.Trim(v, "<>")
		if strings.HasSuffix(v, "@"+fakeRcptDomain) {
			fakeRcptLocal := v[:strings.Index(v, "@")]
			rcptAddr = trueRcptLocal + "+" + fakeRcptLocal + "@" + trueRcptDomain
			break
		}
	}

	// Redirect message
	to := []string{rcptAddr}
	msg.Header["To"] = []string{rcptAddr}

	// Print message for sending
	var hdrs string
	for k, v := range msg.Header {
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
