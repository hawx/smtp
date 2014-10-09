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
	line, err := c.text.ReadLine()
	if err != nil {
		c.t.Fatal(err)
	}
	return line
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

	time.Sleep(time.Millisecond)
	return s
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
			assert.Equal(1, len(msg.Recipients))
			assert.Equal("recipient@example.net", msg.Recipients[0])
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
				assert.Equal(1, len(msg.Recipients))
				assert.Equal("recipient@example.net", msg.Recipients[0])
				assert.Equal("This is the email body\n", msg.Data)
			case 1:
				assert.Equal("sender2@example.org", msg.Sender)
				assert.Equal(1, len(msg.Recipients))
				assert.Equal("recipient2@example.net", msg.Recipients[0])
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
			assert.Equal(3, len(msg.Recipients))
			assert.Equal("recipient1@example.net", msg.Recipients[0])
			assert.Equal("recipient2@example.net", msg.Recipients[1])
			assert.Equal("recipient3@example.net", msg.Recipients[2])
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
			assert.Equal(1, len(msg.Recipients))
			assert.Equal("recipient2@example.net", msg.Recipients[0])
			assert.Equal("This is the email body2\n", msg.Data)
			close(called)
		})

		called2 := make(chan struct{})
		s.Handle(func(msg Message) {
			assert.Equal("sender2@example.org", msg.Sender)
			assert.Equal(1, len(msg.Recipients))
			assert.Equal("recipient2@example.net", msg.Recipients[0])
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

		assert.Equal(c.Verify("sender@example.org").Error(), "502 Command not implemented")

		assert.Nil(c.Quit())
	})
}

func TestExpn(t *testing.T) {
	with(t, func(s Server) {
		c := NewClient(t)

		c.Send("EXPN sender@example.org")
		assert.Equal(t, c.ReadLine(), "502 Command not implemented")
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

func TestVrfy(t *testing.T) {
	with(t, func(s Server) {
		c := NewClient(t)

		c.Send("VRFY some@address.com")
		assert.Equal(t, c.ReadLine(), "502 Command not implemented")
	})
}

func TestRset(t *testing.T) {
	with(t, func(s Server) {
		c := NewClient(t)

		c.Send("RSET")
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

func TestRcpWithoutMail(t *testing.T) {
s := NewServer(t)
	defer s.Close()

	c := NewClient(t)

	c.Send("EHLO local.test")
	c.Skip(2)

	c.Send("RCPT TO:<other.john@example.com>")
	assert.Equal(t, c.ReadLine(), "503 Command out of sequence")
}
