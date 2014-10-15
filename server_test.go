package server

import (
	"github.com/stretchr/testify/assert"
	"net/textproto"
	"fmt"
	"net/smtp"
	"testing"
	"time"
)

const (
	ADDR = "127.0.0.1:9026"
	NAME = "mx.test.server"
	TIMEOUT = 10 * time.Millisecond
)

type Client struct {
	text *textproto.Conn
	t    *testing.T
}

func NewClient(t *testing.T) Client {
	text, err := textproto.Dial("tcp", ADDR)
	if err != nil {
		t.Fatal("NewClient:", err)
	}

	client := Client{text, t}
	assert.Equal(t, client.ReadLine(), "220 " + NAME)
	return client
}

func (c Client) Send(format string, args ...interface{}) {
	if err := c.text.PrintfLine(format, args...); err != nil {
		c.t.Fatal(err)
	}
}

func (c Client) ReadLine() string {
	lines := make(chan string, 1)

	go func() {
		line, err := c.text.ReadLine()
		if err != nil {
			c.t.Fatal(err)
		}
		lines <- line
	}()

	select {
	case line := <-lines:
		return line
	case <-time.After(TIMEOUT):
		return ""
	}
}

func (c Client) Skip(num int) {
	for i := 0; i < num; i++ {
		_, err := c.text.ReadLine()
		if err != nil {
			c.t.Fatal(err)
		}
	}
}

func NewServer(t *testing.T) Server {
	s, err := Listen(ADDR, NAME)
	if err != nil {
		t.Fatal(err)
	}

	return s
}

func NewCatchServer(t *testing.T) (Server, <-chan Message) {
	s := NewServer(t)

	ch := make(chan Message)
	s.Handle(func(m Message) {
		ch <- m
	})

	return s, ch
}

func with(t *testing.T, f func(Server)) {
	s, err := Listen(ADDR, NAME)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond)
	f(s)

	s.Close()
}

func TestSenderRecipientBodyAndQuit(t *testing.T) {
	assert := assert.New(t)

	with(t, func(s Server) {
		called := make(chan struct{})
		s.Handle(func(msg Message) {
			assert.Equal("sender@example.org", msg.Sender)
			if assert.Equal(1, len(msg.Recipients)) {
				assert.Equal("recipient@example.net", msg.Recipients[0])
			}
			assert.Equal("This is the email body\n", msg.Data)
			close(called)
		})

		c, err := smtp.Dial(ADDR)
		assert.Nil(err)

		assert.Nil(c.Mail("sender@example.org"))
		assert.Nil(c.Rcpt("recipient@example.net"))

		wc, err := c.Data()
		assert.Nil(err)
		_, err = fmt.Fprintf(wc, "This is the email body")
		assert.Nil(err)
		assert.Nil(wc.Close())

		assert.Nil(c.Quit())

		select {
		case <-called:
		case <-time.After(time.Second):
			t.Log("timed out")
			t.Fail()
		}
	})
}

func TestSendMultipleMessagesWithSameConnection(t *testing.T) {
	assert := assert.New(t)

	with(t, func(s Server) {
		called := make(chan struct{})
		calls := 0
		s.Handle(func(msg Message) {
			switch calls {
			case 0:
				assert.Equal("sender@example.org", msg.Sender)
				if assert.Equal(1, len(msg.Recipients)) {
					assert.Equal("recipient@example.net", msg.Recipients[0])
				}
				assert.Equal("This is the email body\n", msg.Data)
			case 1:
				assert.Equal("sender2@example.org", msg.Sender)
				if assert.Equal(1, len(msg.Recipients)) {
					assert.Equal("recipient2@example.net", msg.Recipients[0])
				}
				assert.Equal("This is the email body 2\n", msg.Data)
				close(called)
			default:
				t.Fail()
			}

			calls++
		})

		c, err := smtp.Dial(ADDR)
		assert.Nil(err)

		assert.Nil(c.Mail("sender@example.org"))
		assert.Nil(c.Rcpt("recipient@example.net"))

		wc, err := c.Data()
		assert.Nil(err)
		_, err = fmt.Fprintf(wc, "This is the email body")
		assert.Nil(err)
		assert.Nil(wc.Close())

		assert.Nil(c.Mail("sender2@example.org"))
		assert.Nil(c.Rcpt("recipient2@example.net"))

		wc, err = c.Data()
		assert.Nil(err)
		_, err = fmt.Fprintf(wc, "This is the email body 2")
		assert.Nil(err)
		assert.Nil(wc.Close())

		assert.Nil(c.Quit())

		select {
		case <-called:
		case <-time.After(time.Second):
			t.Log("timed out")
			t.Fail()
		}
	})
}

func TestMessageToMultipleRecipients(t *testing.T) {
	assert := assert.New(t)

	with(t, func(s Server) {
		called := make(chan struct{})
		s.Handle(func(msg Message) {
			assert.Equal("sender@example.org", msg.Sender)
			if assert.Equal(3, len(msg.Recipients)) {
				assert.Equal("recipient1@example.net", msg.Recipients[0])
				assert.Equal("recipient2@example.net", msg.Recipients[1])
				assert.Equal("recipient3@example.net", msg.Recipients[2])
			}
			assert.Equal("This is the email body\n", msg.Data)
			close(called)
		})

		c, err := smtp.Dial(ADDR)
		assert.Nil(err)

		assert.Nil(c.Mail("sender@example.org"))
		assert.Nil(c.Rcpt("recipient1@example.net"))
		assert.Nil(c.Rcpt("recipient2@example.net"))
		assert.Nil(c.Rcpt("recipient3@example.net"))

		wc, err := c.Data()
		assert.Nil(err)
		_, err = fmt.Fprintf(wc, "This is the email body")
		assert.Nil(err)
		assert.Nil(wc.Close())

		assert.Nil(c.Quit())

		select {
		case <-called:
		case <-time.After(time.Second):
			t.Log("timed out")
			t.Fail()
		}
	})
}

func TestSenderRecipientBodyAndQuitWithReset(t *testing.T) {
	assert := assert.New(t)

	with(t, func(s Server) {
		called := make(chan struct{})
		s.Handle(func(msg Message) {
			assert.Equal("sender2@example.org", msg.Sender)
			if assert.Equal(1, len(msg.Recipients)) {
				assert.Equal("recipient2@example.net", msg.Recipients[0])
			}
			assert.Equal("This is the email body2\n", msg.Data)
			close(called)
		})

		called2 := make(chan struct{})
		s.Handle(func(msg Message) {
			assert.Equal("sender2@example.org", msg.Sender)
			if assert.Equal(1, len(msg.Recipients)) {
				assert.Equal("recipient2@example.net", msg.Recipients[0])
			}
			assert.Equal("This is the email body2\n", msg.Data)
			close(called2)
		})

		c, err := smtp.Dial(ADDR)
		assert.Nil(err)

		assert.Nil(c.Mail("sender@example.org"))
		assert.Nil(c.Rcpt("recipient@example.net"))

		assert.Nil(c.Reset())

		assert.Nil(c.Mail("sender2@example.org"))
		assert.Nil(c.Rcpt("recipient2@example.net"))

		wc, err := c.Data()
		assert.Nil(err)
		_, err = fmt.Fprintf(wc, "This is the email body2")
		assert.Nil(err)
		assert.Nil(wc.Close())

		assert.Nil(c.Quit())

		select {
		case <-called:
		case <-time.After(time.Second):
			t.Log("timed out")
			t.Fail()
		}

		select {
		case <-called2:
		case <-time.After(time.Second):
			t.Log("timed out2")
			t.Fail()
		}
	})
}

func TestVerify(t *testing.T) {
	assert := assert.New(t)

	with(t, func(s Server) {
		c, err := smtp.Dial(ADDR)
		assert.Nil(err)

		assert.Equal(c.Verify("sender@example.org").Error(), "252 Cannot VRFY user, but will attempt delivery")

		assert.Nil(c.Quit())
	})
}

func TestHelp(t *testing.T) {
	with(t, func(s Server) {
		c := NewClient(t)

		c.Send("HELP")
		assert.Equal(t, c.ReadLine(), "502 Command not implemented")
	})
}

func TestNoop(t *testing.T) {
	with(t, func(s Server) {
		c := NewClient(t)

		c.Send("NOOP")
		assert.Equal(t, c.ReadLine(), "250 Ok")
	})
}

func TestUnrecognizedCommand(t *testing.T) {
	with(t, func(s Server) {
		c := NewClient(t)

		c.Send("LOOK")
		assert.Equal(t, c.ReadLine(), "500 Command unrecognized")
	})
}

// HELO

func TestHelo(t *testing.T) {
	s := NewServer(t)
	defer s.Close()

	c := NewClient(t)

	c.Send("HELO local.test")
	assert.Equal(t, c.ReadLine(), "250 " + NAME + " at your service")
}

func TestHeloWithNoArgument(t *testing.T) {
	s := NewServer(t)
	defer s.Close()

	c := NewClient(t)

	c.Send("HELO")
	assert.Equal(t, c.ReadLine(), "250 " + NAME + " at your service")
}

// EHLO

func TestEhlo(t *testing.T) {
	s := NewServer(t)
	defer s.Close()

	c := NewClient(t)

	c.Send("EHLO local.test")
	assert.Equal(t, c.ReadLine(), "250-" + NAME + " at your service")
	assert.Equal(t, c.ReadLine(), "250 8BITMIME")
}

func TestEhloWithNoArgument(t *testing.T) {
	s := NewServer(t)
	defer s.Close()

	c := NewClient(t)

	c.Send("EHLO")
	assert.Equal(t, c.ReadLine(), "250-" + NAME + " at your service")
	assert.Equal(t, c.ReadLine(), "250 8BITMIME")
}

// MAIL

func TestMail(t *testing.T) {
	s := NewServer(t)
	defer s.Close()

	c := NewClient(t)

	c.Send("EHLO local.test")
	c.Skip(2)

	c.Send("MAIL FROM:<john.doe@example.com>")
	assert.Equal(t, c.ReadLine(), "250 Ok")
}

func TestMailWithNullAddress(t *testing.T) {
	s := NewServer(t)
	defer s.Close()

	c := NewClient(t)

	c.Send("EHLO local.test")
	c.Skip(2)

	c.Send("MAIL FROM:<>")
	assert.Equal(t, c.ReadLine(), "250 Ok")
}

func TestMailWithSyntaxErrors(t *testing.T) {
	s := NewServer(t)
	defer s.Close()

	for _, testCase := range []string{
		"MAIL FROM:<john.doe@example.com",
		"MAIL FROM:",
		"MAIL TO:<john.doe@example.com>",
		"MAIL",
	} {
		c := NewClient(t)

		c.Send("EHLO local.test")
		c.Skip(2)

		c.Send(testCase)
		assert.Equal(t, c.ReadLine(), "501 Syntax error")
	}
}

func TestMailWithoutEhlo(t *testing.T) {
	s := NewServer(t)
	defer s.Close()

	c := NewClient(t)

	c.Send("MAIL FROM:<john.doe@example.com>")
	assert.Equal(t, c.ReadLine(), "503 Command out of sequence")
}

// RCPT

func TestRcpt(t *testing.T) {
	s := NewServer(t)
	defer s.Close()

	c := NewClient(t)

	c.Send("EHLO local.test")
	c.Skip(2)

	c.Send("MAIL FROM:<john.doe@example.com>")
	c.Skip(1)

	c.Send("RCPT TO:<other.john@example.com>")
	assert.Equal(t, c.ReadLine(), "250 Ok")
}

func TestRcptWithSyntaxErrors(t *testing.T) {
	s := NewServer(t)
	defer s.Close()

	for _, testCase := range []string{
		"RCPT TO:<>",
		"RCPT TO:<john.doe@example.com",
		"RCPT TO:",
		"RCPT FROM:<john.doe@example.com",
		"RCPT",
	} {
		c := NewClient(t)

		c.Send("EHLO local.test")
		c.Skip(2)

		c.Send("MAIL FROM:<john.doe@example.com>")
		c.Skip(1)

		c.Send(testCase)
		assert.Equal(t, c.ReadLine(), "501 Syntax error")
	}
}

func TestRcptWithoutMail(t *testing.T) {
	s := NewServer(t)
	defer s.Close()

	c := NewClient(t)

	c.Send("EHLO local.test")
	c.Skip(2)

	c.Send("RCPT TO:<other.john@example.com>")
	assert.Equal(t, c.ReadLine(), "503 Command out of sequence")
}

// DATA

func TestData(t *testing.T) {
	s, ch := NewCatchServer(t)
	defer s.Close()

	c := NewClient(t)

	c.Send("EHLO local.test")
	c.Skip(2)

	c.Send("MAIL FROM:<john.doe@example.com>")
	c.Skip(1)

	c.Send("RCPT TO:<jane.doe@example.org>")
	c.Skip(1)

	c.Send("DATA")
	assert.Equal(t, c.ReadLine(), "354 End data with <CRLF>.<CRLF>")

	c.Send("ok so here is the message")
	c.Send("it goes a bit like this")
	c.Send("that was it")

	c.Send(".")
	assert.Equal(t, c.ReadLine(), "250 Ok")

	select {
	case msg := <-ch:
		assert.Equal(t, msg.Sender, "john.doe@example.com")
		if assert.Equal(t, 1, len(msg.Recipients)) {
			assert.Equal(t, "jane.doe@example.org", msg.Recipients[0])
		}
		assert.Equal(t, "ok so here is the message\nit goes a bit like this\nthat was it\n", msg.Data)
	case <-time.After(TIMEOUT):
		t.Log("timed out")
		t.Fail()
	}
}

func TestDataWithEmptyBody(t *testing.T) {
	s, ch := NewCatchServer(t)
	defer s.Close()

	c := NewClient(t)

	c.Send("EHLO local.test")
	c.Skip(2)

	c.Send("MAIL FROM:<john.doe@example.com>")
	c.Skip(1)

	c.Send("RCPT TO:<jane.doe@example.org>")
	c.Skip(1)

	c.Send("DATA")
	assert.Equal(t, c.ReadLine(), "354 End data with <CRLF>.<CRLF>")

	c.Send(".")
	assert.Equal(t, c.ReadLine(), "250 Ok")

	select {
	case msg := <-ch:
		assert.Equal(t, "john.doe@example.com", msg.Sender)
		if assert.Equal(t, 1, len(msg.Recipients)) {
			assert.Equal(t, "jane.doe@example.org", msg.Recipients[0])
		}
		assert.Equal(t, "", msg.Data)
	case <-time.After(TIMEOUT):
		t.Log("timed out")
		t.Fail()
	}
}

func TestDataWithoutRcpt(t *testing.T) {
	s, ch := NewCatchServer(t)
	defer s.Close()

	c := NewClient(t)

	c.Send("EHLO local.test")
	c.Skip(2)

	c.Send("MAIL FROM:<john.doe@example.com>")
	c.Skip(1)

	c.Send("DATA")
	assert.Equal(t, c.ReadLine(), "503 Command out of sequence")

	select {
	case <-ch:
	  t.Log("Should not have got a message")
  case <-time.After(TIMEOUT):
	}
}

// RSET

func TestRset(t *testing.T) {
	s, ch := NewCatchServer(t)
	defer s.Close()

	c := NewClient(t)

	c.Send("EHLO local.test")
	c.Skip(2)

	c.Send("MAIL FROM:<john.doe@example.com>")
	c.Skip(1)

	c.Send("RCPT TO:<jane.doe@example.org>")
	c.Skip(1)

	c.Send("RSET")
	assert.Equal(t, c.ReadLine(), "250 Ok")

	c.Send("MAIL FROM:<john.doe2@example.com>")
	c.Skip(1)

	c.Send("RCPT TO:<jane.doe2@example.org>")
	c.Skip(1)

	c.Send("DATA")
	assert.Equal(t, c.ReadLine(), "354 End data with <CRLF>.<CRLF>")

	c.Send("that was it")
	c.Send(".")
	assert.Equal(t, c.ReadLine(), "250 Ok")

	select {
	case msg := <-ch:
		assert.Equal(t, msg.Sender, "john.doe2@example.com")
		if assert.Equal(t, 1, len(msg.Recipients)) {
			assert.Equal(t, "jane.doe2@example.org", msg.Recipients[0])
		}
		assert.Equal(t, "that was it\n", msg.Data)
	case <-time.After(TIMEOUT):
		t.Log("timed out")
		t.Fail()
	}
}

// VRFY

func TestVrfy(t *testing.T) {
	s := NewServer(t)
	defer s.Close()

	c := NewClient(t)

	c.Send("VRFY john.doe@example.com")
	assert.Equal(t, c.ReadLine(), "252 Cannot VRFY user, but will attempt delivery")
}

func TestVrfyWithImplementation(t *testing.T) {
	s := NewServer(t)
	defer s.Close()

	s.Verify(func(addr string) Mailbox {
		if addr == "john.doe@example.com" || addr == "john.doe" {
			return Mailbox{"John Doe", "john.doe@example.com"}
		}

		return Mailbox{}
	})

	c := NewClient(t)

	c.Send("VRFY john.doe@example.com")
	assert.Equal(t, c.ReadLine(), "250 John Doe <john.doe@example.com>")

	c.Send("VRFY john.doe")
	assert.Equal(t, c.ReadLine(), "250 John Doe <john.doe@example.com>")

	c.Send("VRFY jane.doe@example.com")
	assert.Equal(t, c.ReadLine(), "252 Cannot VRFY user, but will attempt delivery")
}

// EXPN

func TestExpn(t *testing.T) {
	s := NewServer(t)
	defer s.Close()

	c := NewClient(t)

	c.Send("EXPN Cool-List")
	assert.Equal(t, c.ReadLine(), "550 Access denied")
}

func TestExpnWithImplementation(t *testing.T) {
	s := NewServer(t)
	defer s.Close()

	s.Expand(func(addr string) []Mailbox {
		if addr != "Those-Does" && addr != "Those-Does@example.com" {
			return []Mailbox{}
		}

		return []Mailbox{
			Mailbox{"John Doe", "john.doe@example.com"},
			Mailbox{"Jane Doe", "jane.doe@example.com"},
		}
	})

	c := NewClient(t)

	c.Send("EXPN Those-Does")
	assert.Equal(t, c.ReadLine(), "250-John Doe <john.doe@example.com>")
	assert.Equal(t, c.ReadLine(), "250 Jane Doe <jane.doe@example.com>")

	c.Send("EXPN Those-Does@example.com")
	assert.Equal(t, c.ReadLine(), "250-John Doe <john.doe@example.com>")
	assert.Equal(t, c.ReadLine(), "250 Jane Doe <jane.doe@example.com>")
}
