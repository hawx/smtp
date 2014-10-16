package smtp

import (
	"fmt"
	"crypto/rand"
	"crypto/hmac"
	"crypto/md5"
	"encoding/base64"
)

type Authenticator interface {
	Start() (toClient string, err error)
	Auth(user, fromClient string) (ok bool)
}

func CramAuthenticator(a func(string) string) Authenticator {
	return &cramAuthenticator{a: a}
}

type cramAuthenticator struct {
	a    func(string) string
	cont []byte
}

func (c *cramAuthenticator) Start() (toClient string, err error) {
	c.cont = make([]byte, 52)
	_, err = rand.Read(c.cont)
	if err != nil {
		return
	}

	toClient = base64.StdEncoding.EncodeToString(c.cont)
	return
}

func (c *cramAuthenticator) Auth(user, fromClient string) bool {
	secret := c.a(user)
	d := hmac.New(md5.New, []byte(secret))
	d.Write(c.cont)
	sum := fmt.Sprintf("%x", d.Sum(make([]byte, 0, d.Size())))

	return sum == fromClient
}
