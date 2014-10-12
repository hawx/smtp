package server

import (
	"net/textproto"
	"regexp"
	"log"
)

const (
	OK = "250 Ok"
	BYE = "221 Bye"
	END_DATA_WITH = "354 End data with <CRLF>.<CRLF>"
	COMMAND_UNRECOGNIZED = "500 Command unrecognized"
	SYNTAX_ERROR = "501 Syntax error"
	COMMAND_NOT_IMPLEMENTED = "502 Command not implemented"
	OUT_OF_SEQUENCE = "503 Command out of sequence"
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
		write(text, SYNTAX_ERROR)
		return transaction
	}

	if newTransaction, ok := transaction.Sender(matches[1]); ok {
		write(text, OK)
		return newTransaction
	} else {
		write(text, OUT_OF_SEQUENCE)
		return transaction
	}
}

func rcpt(args string, text *textproto.Conn, transaction Transaction) Transaction {
	matches := rcptRe.FindStringSubmatch(args)
	if matches == nil || len(matches) != 2 {
		write(text, SYNTAX_ERROR)
		return transaction
	}

	if newTransaction, ok := transaction.Recipient(matches[1]); ok {
		write(text, OK)
		return newTransaction
	} else {
		write(text, OUT_OF_SEQUENCE)
		return transaction
	}
}

func data(text *textproto.Conn, transaction Transaction) (Message, bool) {
	if _, ok := transaction.Data("test"); !ok {
		write(text, OUT_OF_SEQUENCE)
		return Message{}, false
	}

	write(text, END_DATA_WITH)

	d, err := text.ReadDotBytes()
	if err != nil {
		log.Println("DATA:", err)
		return Message{}, false
	}

	text.PrintfLine(OK)
	message, _ := transaction.Data(string(d))
	return message, true
}

func rset(text *textproto.Conn) {
	write(text, OK)
}

func quit(text *textproto.Conn) {
	write(text, BYE)
}
