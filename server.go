package server

import (
	"errors"
	"log"
	"net"
	"net/textproto"
	"regexp"
	"strings"
)

type Handler func(Message)

type Server interface {
	Handle(Handler)
	Close() error
}

type server struct {
	name     string
	ln       net.Listener
	out      chan Message
	handlers []Handler
	quit     chan struct{}
}

func Listen(addr, name string) (Server, error) {
	tcp, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	s := &server{
		name:     name,
		ln:       tcp,
		handlers: []Handler{},
		out:      make(chan Message),
		quit:     make(chan struct{}),
	}

	go s.start()
	go s.handle()

	return s, nil
}

func (s *server) Handle(handler Handler) {
	s.handlers = append(s.handlers, handler)
}

func (s *server) Close() error {
	close(s.quit)
	return s.ln.Close()
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
		s.serve(text)

		conn.Close()
	}
}

func (s *server) handle() {
	for {
		msg := <-s.out
		for _, handler := range s.handlers {
			handler(msg)
		}
	}
}

func (s *server) serve(text *textproto.Conn) {
	text.PrintfLine("220 %s", s.name)

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

	s.out <- transaction.Message()
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
		helo(cmd, rest, text)
	}

	write(text, "250 Hello %s, I am glad to meet you", rest)
	return nil
}

func helo(cmd, rest string, text *textproto.Conn) error {
	if cmd != "HELO" {
		return errors.New("Not HELO")
	}

	write(text, "250 Hello %s, I am glad to meet you", rest)
	return nil
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
