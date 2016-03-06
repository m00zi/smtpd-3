package main

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"os"
	"strings"
)

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
	session, err := smtpHandle(conn)
	if err != nil && err != io.EOF {
		log.Print(err)
		return
	}
	if len(session.data) == 0 {
		return
	}
	if err := sendMail(session); err != nil {
		log.Print(err)
	} else {
		log.Print("mail sent")
	}
}

func smtpHandle(conn net.Conn) (*smtpSession, error) {
	defer conn.Close()
	log.Printf("Handling %+v", conn.RemoteAddr())
	defer log.Printf("Closing %+v", conn.RemoteAddr())

	buf := &bytes.Buffer{}
	r := textproto.NewReader(bufio.NewReader(io.TeeReader(conn, buf)))
	w := textproto.NewWriter(bufio.NewWriter(io.MultiWriter(conn, buf)))
	defer io.Copy(os.Stdout, buf)

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

func sendMail(s *smtpSession) error {
	auth := smtp.PlainAuth("", smtpUsername, smtpPassword, smtpServer)
	parsed, err := mail.ParseAddress(s.mailFrom)
	if err != nil {
		return err
	}
	from := parsed.Address

	// Compute recipient address
	var rcptAddr string
	for _, v := range s.rcptTo {
		parsed, err := mail.ParseAddress(v)
		if err != nil {
			return err
		}
		addr := strings.Trim(parsed.Address, "<>")
		if strings.HasSuffix(addr, "@"+fakeRcptDomain) {
			fakeRcptLocal := addr[:strings.Index(addr, "@")]
			rcptAddr = trueRcptLocal + "+" + fakeRcptLocal + "@" + trueRcptDomain
			break
		}
	}

	// Redirect message
	to := []string{rcptAddr}

	return smtp.SendMail(smtpServer+":"+smtpPort, auth, from, to, []byte(strings.Join(s.data, "\r\n")))
}
