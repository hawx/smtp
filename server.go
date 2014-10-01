package server

import (
	"errors"
	"log"
	"net"
	"net/textproto"
	"strings"
	"regexp"
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
		cmd, rest, err := read(text)
		if err != nil {
			log.Println(err)
			return
		}

		switch strings.ToUpper(cmd) {
		case "MAIL":
			sender, err := mail(rest, text)
			if err != nil {
				log.Println(err)
				return
			}

			transaction.Sender(sender)

		case "RCPT":
			recipient, err := rcpt(rest, text)
			if err != nil {
				log.Println(err)
				return
			}

			transaction.Recipient(recipient)

		case "DATA":
			d, err := data(text)
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
			quit(text)
			break loop

		default:
			log.Println(cmd, "not recognised")
		}
	}

	out <- transaction.Message()
}

func read(text *textproto.Conn) (string, string, error) {
	line, err := text.ReadLine()
	if err != nil {
		return "", "", err
	}

	parts := strings.SplitN(line, " ", 2)
	if len(parts) == 1 {
		return parts[0], "", nil
	}

	return parts[0], parts[1], nil
}

func write(text *textproto.Conn, format string, args ...interface{}) {
	text.PrintfLine(format, args)
}

func ehlo(text *textproto.Conn) error {
	cmd, rest, err := read(text)
	if err != nil {
		return err
	}

	if cmd != "EHLO" {
		return errors.New("Not EHLO")
	}

	write(text, "250 Hello %s, I am glad to meet you", rest)
	return nil
}

func parseAddress(address string) (string, error) {
	if address[0] != '<' && address[len(address)-1] != '>' {
		return "", errors.New("Address must be between '<' and '>'")
	}

	return address[1:len(address)-1], nil
}

var (
	mailRe = regexp.MustCompile("FROM:<(.*?)>")
	rcptRe = regexp.MustCompile("TO:<(.*?)>")
)

func mail(args string, text *textproto.Conn) (string, error) {
	matches := mailRe.FindStringSubmatch(args)
	if matches == nil || len(matches) != 2 {
		return "", errors.New("MAIL: No address")
	}

	write(text, "250 Ok")
	return matches[1], nil
}

func rcpt(args string, text *textproto.Conn) (string, error) {
	matches := rcptRe.FindStringSubmatch(args)
	if matches == nil || len(matches) != 2 {
		return "", errors.New("RCPT: No address")
	}

	write(text, "250 Ok")
	return matches[1], nil
}

func data(text *textproto.Conn) (string, error) {
	write(text, "354 End data with <CRLF>.<CRLF>")

	d, err := text.ReadDotBytes()
	if err != nil {
		return "", err
	}

	text.PrintfLine("250 Ok")
	return string(d), nil
}

func rset(text *textproto.Conn) error {
	write(text, "250 Ok")
	return nil
}

func vrfy(text *textproto.Conn) error {
	write(text, "502 Command not implemented")
	return nil
}

func quit(text *textproto.Conn) error {
	write(text, "221 Bye")
	return nil
}
