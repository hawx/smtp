package server

import (
	"github.com/stretchr/testify/assert"
	"fmt"
	"net/smtp"
	"testing"
	"time"
)

const ADDR = "127.0.0.1:9025"

func with(t *testing.T, f func(Server)) {
	s, err := Listen(ADDR)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond)
	f(s)

	s.Close()
}

func TestConnect(t *testing.T) {
	assert := assert.New(t)

	with(t, func(s Server) {
		_, err := smtp.Dial(ADDR)
		assert.Nil(err)
	})
}

func TestSender(t *testing.T) {
	assert := assert.New(t)

	with(t, func(s Server) {
		c, err := smtp.Dial(ADDR)
		assert.Nil(err)

		assert.Nil(c.Mail("sender@example.org"))
	})
}

func TestSenderAndRecipient(t *testing.T) {
	assert := assert.New(t)

	with(t, func(s Server) {
		c, err := smtp.Dial(ADDR)
		assert.Nil(err)

		assert.Nil(c.Mail("sender@example.org"))
		assert.Nil(c.Rcpt("recipient@example.net"))
	})
}

func TestSenderRecipientAndBody(t *testing.T) {
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
	})
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

		assert.Nil(c.Quit())

		select {
		case msg := <-s.Out():
			assert.Equal("sender@example.org", msg.Sender)
			assert.Equal(1, len(msg.Recipients))
			assert.Equal("recipient@example.net", msg.Recipients[0])
			assert.Equal("This is the email body\n", msg.Data)
		case <-time.After(time.Second):
			t.Log("timed out")
			t.Fail()
		}
	})
}
