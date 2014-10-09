package server

import (
	"log"
	"io"
	"net"
	"net/textproto"
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
				log.Println("start:", err)
				continue
			}
		}

		text := textproto.NewConn(conn)
		go s.serve(text, conn)
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

func (s *server) serve(text *textproto.Conn, closer io.Closer) {
	defer closer.Close()
	text.PrintfLine("220 %s", s.name)

firstloop:
	for {
		cmd, _, err := read(text)
		if err != nil {
			if err == io.EOF {
				return
			}

			log.Println("read:", err)
			return
		}

		switch strings.ToUpper(cmd) {
		case "EHLO":
			ehlo(s.name, text)
			break firstloop

		case "HELO":
			helo(s.name, text)
			break firstloop

		case "NOOP", "RSET":
			write(text, "250 Ok")

		case "VRFY", "EXPN", "HELP":
			write(text, "502 Command not implemented")

		case "MAIL":
			write(text, "503 Command out of sequence")

		default:
			write(text, "500 Command unrecognized")
		}
	}

	transaction := NewTransaction()

loop:
	for {
		cmd, rest, err := read(text)
		if err != nil {
			if err == io.EOF {
				return
			}

			log.Println("read:", err)
			return
		}

		switch strings.ToUpper(cmd) {
		case "MAIL":
			sender := mail(rest, text)

			transaction = NewTransaction()
			transaction.Sender(sender)

		case "RCPT":
			if !transaction.HasSender() {
				write(text, "503 Command out of sequence")
				continue
			}

			recipient := rcpt(rest, text)
			transaction.Recipient(recipient)

		case "DATA":
			d, err := data(text)
			if err != nil {
				log.Println("DATA:", err)
				continue
			}

			transaction.Data(d)
			s.out <- transaction.Message()
			transaction = NewTransaction()

		case "RSET":
			rset(text)
			transaction = NewTransaction()

		case "QUIT":
			quit(text)
			break loop

		case "NOOP":
			write(text, "250 Ok")

		case "VRFY", "EXPN", "HELP":
			write(text, "502 Command not implemented")

		default:
			write(text, "500 Command unrecognized")
		}
	}
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
	text.PrintfLine(format, args...)
}
