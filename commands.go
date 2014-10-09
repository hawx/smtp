package server

import (
	"net/textproto"
	"regexp"
	"log"
)

var (
	mailRe = regexp.MustCompile("FROM:<(.*?)>")
	rcptRe = regexp.MustCompile("TO:<(.+)>")
)

func ehlo(name string, text *textproto.Conn) {
	write(text, "250-%s at your service", name)
	write(text, "250 8BITMIME")
}

func helo(name string, text *textproto.Conn) {
	write(text, "250 %s at your service", name)
}

func mail(args string, text *textproto.Conn, transaction Transaction) Transaction {
	matches := mailRe.FindStringSubmatch(args)
	if matches == nil || len(matches) != 2 {
		write(text, "501 Syntax error")
		return transaction
	}

	if newTransaction, ok := transaction.Sender(matches[1]); ok {
		write(text, "250 Ok")
		return newTransaction
	} else {
		write(text, "503 Command out of sequence")
		return transaction
	}
}

func rcpt(args string, text *textproto.Conn, transaction Transaction) Transaction {
	matches := rcptRe.FindStringSubmatch(args)
	if matches == nil || len(matches) != 2 {
		write(text, "501 Syntax error")
		return transaction
	}

	if newTransaction, ok := transaction.Recipient(matches[1]); ok {
		write(text, "250 Ok")
		return newTransaction
	} else {
		write(text, "503 Command out of sequence")
		return transaction
	}
}

func data(text *textproto.Conn) (string, error) {
	write(text, "354 End data with <CRLF>.<CRLF>")

	d, err := text.ReadDotBytes()
	if err != nil {
		log.Println("DATA:", err)
		return "", err
	}

	text.PrintfLine("250 Ok")
	return string(d), nil
}

func rset(text *textproto.Conn) {
	write(text, "250 Ok")
}

func quit(text *textproto.Conn) {
	write(text, "221 Bye")
}
