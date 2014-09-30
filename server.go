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
	ln    net.Listener
	out   chan Message
	quit  chan struct{}
}

func Listen(addr string) (Server, error) {
	tcp, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	s := &server{
	  ln:   tcp,
	  out:  make(chan Message),
	  quit: make(chan struct{}),
	}

	go s.start()
	return s, nil
}

func (s *server) start() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
				log.Println(err)
				continue
			}
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
	close(s.quit)
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
			return
		}

		switch strings.ToUpper(parts[0]) {
		case "MAIL":
			sender, err := mail(parts, text)
			if err != nil {
				log.Println(err)
				return
			}

			transaction.Sender(sender)

		case "RCPT":
			recipient, err := rcpt(parts, text)
			if err != nil {
				log.Println(err)
				return
			}

			transaction.Recipient(recipient)

		case "DATA":
			d, err := data(parts, text)
			if err != nil {
				log.Println(err)
				return
			}

			transaction.Data(d)

		case "RSET":
			if err := rset(text); err != nil {
				log.Println(err)
				return
			}
			transaction = NewTransaction()

		case "VRFY":
			if err := vrfy(text); err != nil {
				log.Println(err)
				return
			}

		case "QUIT":
			quit(parts, text)
			break loop

		default:
			log.Println(parts[0], "not recognised")
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

func rset(text *textproto.Conn) error {
	write(text, "250 Ok")
	return nil
}

func vrfy(text *textproto.Conn) error {
	write(text, "502 Command not implemented")
	return nil
}
