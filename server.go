// Package smtp provides an SMTP server for receiving mail.
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

// User represents an account that can receive mail with a name and address
// usually formatted as, e.g., "John Doe <john.doe@example.com>".
type User struct {
	Name, Addr string
}

// A Handler receives Messages when transactions are completed.
type Handler func(Message)

// A Verifier verifies whether its argument represents a user or email address
// on the system, and if so returns the details as a User; otherwise an empty
// User is returned.
type Verifier func(string) User

// An Expander expands mailing lists into the list of Users who are in it.
type Expander func(string) []User

type Server struct {
	name     string
	ln       net.Listener
	out      chan Message
	quit     chan struct{}

	handlers []Handler
	verifier Verifier
	expander Expander
}

// Listen creates a new Server listening at the local network address laddr and
// will announce itself to new connections with the name given.
func Listen(laddr, name string) (*Server, error) {
	tcp, err := net.Listen("tcp", laddr)
	if err != nil {
		return nil, err
	}

	s := &Server{
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

// Handle registers a new Handler to the Server. All Handlers will be run for
// each Message received, on completion of a mail transaction.
func (s *Server) Handle(handler Handler) {
	s.handlers = append(s.handlers, handler)
}

// Verify registers the Verifier to be used when a VRFY command is issued to the
// Server. If a Verifier was previously registered it is overwritten.
func (s *Server) Verify(verifier Verifier) {
	s.verifier = verifier
}

// Expand registers the Expander to be used when an EXPN command is issued to
// the Server. If an Expander was previously registered it is overwritten.
func (s *Server) Expand(expander Expander) {
	s.expander = expander
}

// Close stops the Server from accepting new connections and listening.
func (s *Server) Close() error {
	// TODO: Make sure Close() kills in-progress transactions.
	close(s.quit)
	return s.ln.Close()
}

func (s *Server) start() {
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

		go s.serve(newConn(conn), conn)
	}
}

func (s *Server) handle() {
	for {
		msg := <-s.out
		for _, handler := range s.handlers {
			handler(msg)
		}
	}
}

func (s *Server) serve(text connection, closer io.Closer) {
	defer closer.Close()

	text.write("220 %s", s.name)
	transaction := newTransaction()

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
			transaction = resetTransaction(transaction)
			text.write("250-%s at your service", s.name)
			text.write("250 8BITMIME")

		case "HELO":
			transaction = resetTransaction(transaction)
			text.write("250 %s at your service", s.name)

		case "MAIL":
			transaction = mail(rest, text, transaction)

		case "RCPT":
			transaction = rcpt(rest, text, transaction)

		case "DATA":
			if message, ok := data(text, transaction); ok {
				s.out <- message
				transaction = resetTransaction(transaction)
			}

		case "RSET":
			transaction = resetTransaction(transaction)
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
