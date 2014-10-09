package server

import (
	"net/textproto"
	"regexp"
	"fmt"
)

var (
	mailRe = regexp.MustCompile("FROM:<(.*?)>")
	rcptRe = regexp.MustCompile("TO:<(.+)>")
)

func ehlo(name string, text *textproto.Conn) error {
	write(text, "250-%s at your service", name)
	write(text, "250 8BITMIME")
	return nil
}

func helo(name string, text *textproto.Conn) error {
	write(text, "250 %s at your service", name)
	return nil
}

func mail(args string, text *textproto.Conn) (string, error) {
	matches := mailRe.FindStringSubmatch(args)
	if matches == nil || len(matches) != 2 {
		write(text, "501 Syntax error")
		return "", fmt.Errorf("No address in '%s'", args)
	}

	write(text, "250 Ok")
	return matches[1], nil
}

func rcpt(args string, text *textproto.Conn) (string, error) {
	matches := rcptRe.FindStringSubmatch(args)
	if matches == nil || len(matches) != 2 {
		write(text, "501 Syntax error")
		return "", fmt.Errorf("No address in '%s'", args)
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

func quit(text *textproto.Conn) error {
	write(text, "221 Bye")
	return nil
}
