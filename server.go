package server

import (
	"io"
	"log"
	"net"
	"net/textproto"
	"strings"
)

const (
	rOK = "250 Ok"
	rBYE = "221 Bye"
	rEND_DATA_WITH = "354 End data with <CRLF>.<CRLF>"
	rCOMMAND_UNRECOGNIZED = "500 Command unrecognized"
	rSYNTAX_ERROR = "501 Syntax error"
	rCOMMAND_NOT_IMPLEMENTED = "502 Command not implemented"
	rOUT_OF_SEQUENCE = "503 Command out of sequence"
)

type User struct {
	Name, Addr string
}

type Handler func(Message)
type Verifier func(string) User
type Expander func(string) []User

type Server interface {
	Handle(Handler)
	Verify(Verifier)
	Expand(Expander)
	Close() error
}

type server struct {
	name     string
	ln       net.Listener
	out      chan Message
	quit     chan struct{}

	handlers []Handler
	verifier Verifier
	expander Expander
}

func Listen(addr, name string) (Server, error) {
	tcp, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	s := &server{
		name:     name,
		ln:       tcp,
		out:      make(chan Message),
		quit:     make(chan struct{}),
	  handlers: []Handler{},
	  verifier: func(_ string) User {
			return User{}
		},
	  expander: func(_ string) []User {
			return []User{}
		},
	}

	go s.start()
	go s.handle()

	return s, nil
}

func (s *server) Handle(handler Handler) {
	s.handlers = append(s.handlers, handler)
}

func (s *server) Verify(verifier Verifier) {
	s.verifier = verifier
}

func (s *server) Expand(expander Expander) {
	s.expander = expander
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

	write(text, "220 %s", s.name)
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
		case "EHLO":
			transaction = Reset(transaction)
			write(text, "250-%s at your service", s.name)
			write(text, "250 8BITMIME")

		case "HELO":
			transaction = Reset(transaction)
			write(text, "250 %s at your service", s.name)

		case "MAIL":
			transaction = mail(rest, text, transaction)

		case "RCPT":
			transaction = rcpt(rest, text, transaction)

		case "DATA":
			if message, ok := data(text, transaction); ok {
				s.out <- message
				transaction = Reset(transaction)
			}

		case "RSET":
			transaction = Reset(transaction)
			write(text, rOK)

		case "VRFY":
			if box := s.verifier(rest); box != (User{}) {
				write(text, "250 %s <%s>", box.Name, box.Addr)
				continue
			}

			write(text, "252 Cannot VRFY user, but will attempt delivery")

		case "EXPN":
			if boxes := s.expander(rest); len(boxes) > 0 {
				for i, box := range boxes {
					if i == len(boxes) - 1 {
						write(text, "250 %s <%s>", box.Name, box.Addr)
					} else {
						write(text, "250-%s <%s>", box.Name, box.Addr)
					}
				}
				continue
			}

			write(text, "550 Access denied")

		case "QUIT":
			write(text, rBYE)
			break loop

		case "NOOP":
			write(text, rOK)

		case "HELP":
			write(text, rCOMMAND_NOT_IMPLEMENTED)

		default:
			write(text, rCOMMAND_UNRECOGNIZED)
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
