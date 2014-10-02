package server

import (
	"net/textproto"
	"errors"
	"regexp"
)

var (
	mailRe = regexp.MustCompile("FROM:<(.*?)>")
	rcptRe = regexp.MustCompile("TO:<(.*?)>")
)

func ehlo(text *textproto.Conn) error {
	cmd, rest, err := read(text)
	if err != nil {
		return err
	}

	if cmd != "EHLO" {
		helo(cmd, rest, text)
	}

	write(text, "250 Hello %s, I am glad to meet you", rest)
	return nil
}

func helo(cmd, rest string, text *textproto.Conn) error {
	if cmd != "HELO" {
		return errors.New("Not HELO")
	}

	write(text, "250 Hello %s, I am glad to meet you", rest)
	return nil
}

func mail(args string, text *textproto.Conn) (string, error) {
	matches := mailRe.FindStringSubmatch(args)
	if matches == nil || len(matches) != 2 {
		return "", errors.New("MAIL: No address")
	}

	write(text, "250 Ok")
	return matches[1], nil
}

func rcpt(args string, text *textproto.Conn) (string, error) {
	matches := rcptRe.FindStringSubmatch(args)
	if matches == nil || len(matches) != 2 {
		return "", errors.New("RCPT: No address")
	}

	write(text, "250 Ok")
	return matches[1], nil
}

func data(text *textproto.Conn) (string, error) {
	write(text, "354 End data with <CRLF>.<CRLF>")

	d, err := text.ReadDotBytes()
	if err != nil {
		return "", err
	}

	text.PrintfLine("250 Ok")
	return string(d), nil
}

func rset(text *textproto.Conn) error {
	write(text, "250 Ok")
	return nil
}

func vrfy(text *textproto.Conn) error {
	write(text, "502 Command not implemented")
	return nil
}

func quit(text *textproto.Conn) error {
	write(text, "221 Bye")
	return nil
}
