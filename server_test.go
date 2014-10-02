package server

import (
	"github.com/stretchr/testify/assert"
	"fmt"
	"net/smtp"
	"testing"
	"time"
)

const (
	ADDR = "127.0.0.1:9026"
	NAME = "mx.test.server"
)

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
		c, err := smtp.Dial(ADDR)
		assert.Nil(err)

		assert.Nil(c.Mail("sender@example.org"))
		assert.Nil(c.Rcpt("recipient@example.net"))

		wc, err := c.Data()
		assert.Nil(err)
		_, err = fmt.Fprintf(wc, "This is the email body")
		assert.Nil(err)
		assert.Nil(wc.Close())

		called := make(chan struct{})
		s.Handle(func(msg Message) {
			assert.Equal("sender@example.org", msg.Sender)
			assert.Equal(1, len(msg.Recipients))
			assert.Equal("recipient@example.net", msg.Recipients[0])
			assert.Equal("This is the email body\n", msg.Data)
			close(called)
		})

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
		c, err := smtp.Dial(ADDR)
		assert.Nil(err)

		assert.Nil(c.Mail("sender@example.org"))
		assert.Nil(c.Rcpt("recipient@example.net"))

		wc, err := c.Data()
		assert.Nil(err)
		_, err = fmt.Fprintf(wc, "This is the email body")
		assert.Nil(err)
		assert.Nil(wc.Close())

		assert.Nil(c.Reset())

		assert.Nil(c.Mail("sender2@example.org"))
		assert.Nil(c.Rcpt("recipient2@example.net"))

		wc, err = c.Data()
		assert.Nil(err)
		_, err = fmt.Fprintf(wc, "This is the email body2")
		assert.Nil(err)
		assert.Nil(wc.Close())

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

		assert.Equal(c.Verify("sender@example.org").Error(), "502 Command not implemented%!(EXTRA []interface {}=[])")

		assert.Nil(c.Quit())
	})
}
