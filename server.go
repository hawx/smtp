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

	if err := ehlo(text); err != nil {
		log.Println("EHLO:", err)
		return
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
			sender, err := mail(rest, text)
			if err != nil {
				log.Println("MAIL:", err)
				return
			}

			transaction.Sender(sender)

		case "RCPT":
			recipient, err := rcpt(rest, text)
			if err != nil {
				log.Println("RCPT:", err)
				return
			}

			transaction.Recipient(recipient)

		case "DATA":
			d, err := data(text)
			if err != nil {
				log.Println("DATA:", err)
				return
			}

			transaction.Data(d)

		case "RSET":
			if err := rset(text); err != nil {
				log.Println("RSET:", err)
				return
			}
			transaction = NewTransaction()

		case "VRFY":
			if err := vrfy(text); err != nil {
				log.Println("VRFY:", err)
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
