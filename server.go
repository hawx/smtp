package smtp

import (
	"io"
	"log"
	"net"
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

		go s.serve(NewConn(conn), conn)
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

func (s *server) serve(text connection, closer io.Closer) {
	defer closer.Close()

	text.write("220 %s", s.name)
	transaction := NewTransaction()

loop:
	for {
		cmd, rest, err := text.read()
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
			text.write("250-%s at your service", s.name)
			text.write("250 8BITMIME")

		case "HELO":
			transaction = Reset(transaction)
			text.write("250 %s at your service", s.name)

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
			text.write(rOK)

		case "VRFY":
			if box := s.verifier(rest); box != (User{}) {
				text.write("250 %s <%s>", box.Name, box.Addr)
				continue
			}

			text.write("252 Cannot VRFY user, but will attempt delivery")

		case "EXPN":
			if boxes := s.expander(rest); len(boxes) > 0 {
				for i, box := range boxes {
					if i == len(boxes) - 1 {
						text.write("250 %s <%s>", box.Name, box.Addr)
					} else {
						text.write("250-%s <%s>", box.Name, box.Addr)
					}
				}
				continue
			}

			text.write("550 Access denied")

		case "QUIT":
			text.write(rBYE)
			break loop

		case "NOOP":
			text.write(rOK)

		case "HELP":
			text.write(rCOMMAND_NOT_IMPLEMENTED)

		default:
			text.write(rCOMMAND_UNRECOGNIZED)
		}
	}
}
