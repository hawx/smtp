package server

import (
	"errors"
	"log"
	"net"
	"net/textproto"
	"strings"
)

type Server interface {
	Out()   <-chan Message
	Close() error
}

type server struct {
	addr  string
	out   chan Message
	ln    net.Listener
}

func Listen(addr string) (Server, error) {
	s := &server{addr: addr, out: make(chan Message)}

	tcp, err := net.Listen("tcp", s.addr)
	if err != nil {
		return nil, err
	}
	s.ln = tcp

	go s.start()
	return s, nil
}

func (s *server) start() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		text := textproto.NewConn(conn)
		serve(text, s.out)

		conn.Close()
	}
}

func (s *server) Out() <-chan Message {
	return s.out
}

func (s *server) Close() error {
	return s.ln.Close()
}

func serve(text *textproto.Conn, out chan Message) {
	text.PrintfLine("220 what")

	if err := ehlo(text); err != nil {
		log.Println(err)
		return
	}

	transaction := NewTransaction()

loop:
	for {
		parts, err := read(text)
		if err != nil {
			log.Println(err)
			break loop
		}

		switch parts[0] {
		case "MAIL":
			sender, err := mail(parts, text)
			if err != nil {
				log.Println(err)
				break loop
			}

			transaction.Sender(sender)

		case "RCPT":
			recipient, err := rcpt(parts, text)
			if err != nil {
				log.Println(err)
				break loop
			}

			transaction.Recipient(recipient)

		case "DATA":
			d, err := data(parts, text)
			if err != nil {
				log.Println(err)
				break loop
			}

			transaction.Data(d)

		case "QUIT":
			quit(parts, text)
			break loop
		}
	}

	out <- transaction.Message()
}

func read(text *textproto.Conn) ([]string, error) {
	line, err := text.ReadLine()
	if err != nil {
		return []string{}, err
	}

	return strings.Split(line, " "), nil
}

func write(text *textproto.Conn, format string, args ...interface{}) {
	text.PrintfLine(format, args)
}

func ehlo(text *textproto.Conn) error {
	parts, err := read(text)
	if err != nil {
		return err
	}

	if parts[0] != "EHLO" {
		return errors.New("Not EHLO")
	}

	write(text, "250 Hello %s, I am glad to meet you", parts[1])
	return nil
}

func parseAddress(address string) (string, error) {
	if address[0] != '<' && address[len(address)-1] != '>' {
		return "", errors.New("Address must be between '<' and '>'")
	}

	return address[1:len(address)-1], nil
}

func mail(parts []string, text *textproto.Conn) (string, error) {
	parts = strings.SplitN(parts[1], ":", 2)
	if len(parts) < 2 {
		return "", errors.New("MAIL: No address")
	}

	write(text, "250 Ok")
	return parseAddress(parts[1])
}

func rcpt(parts []string, text *textproto.Conn) (string, error) {
	parts = strings.SplitN(parts[1], ":", 2)
	if len(parts) < 2 {
		return "", errors.New("No address in RCPT")
	}

	write(text, "250 Ok")
	return parseAddress(parts[1])
}

func data(parts []string, text *textproto.Conn) (string, error) {
	write(text, "354 End data with <CRLF>.<CRLF>")

	d, err := text.ReadDotBytes()
	if err != nil {
		return "", err
	}

	text.PrintfLine("250 Ok")
	return string(d), nil
}

func quit(parts []string, text *textproto.Conn) error {
	write(text, "221 Bye")
	return nil
}
