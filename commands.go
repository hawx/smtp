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

func mail(args string, text *textproto.Conn, transaction Transaction) Transaction {
	matches := mailRe.FindStringSubmatch(args)
	if matches == nil || len(matches) != 2 {
		write(text, rSYNTAX_ERROR)
		return transaction
	}

	if newTransaction, ok := transaction.Sender(matches[1]); ok {
		write(text, rOK)
		return newTransaction
	} else {
		write(text, rOUT_OF_SEQUENCE)
		return transaction
	}
}

func rcpt(args string, text *textproto.Conn, transaction Transaction) Transaction {
	matches := rcptRe.FindStringSubmatch(args)
	if matches == nil || len(matches) != 2 {
		write(text, rSYNTAX_ERROR)
		return transaction
	}

	if newTransaction, ok := transaction.Recipient(matches[1]); ok {
		write(text, rOK)
		return newTransaction
	} else {
		write(text, rOUT_OF_SEQUENCE)
		return transaction
	}
}

func data(text *textproto.Conn, transaction Transaction) (Message, bool) {
	if _, ok := transaction.Data("test"); !ok {
		write(text, rOUT_OF_SEQUENCE)
		return Message{}, false
	}

	write(text, rEND_DATA_WITH)

	d, err := text.ReadDotBytes()
	if err != nil {
		log.Println("DATA:", err)
		return Message{}, false
	}

	text.PrintfLine(rOK)
	message, _ := transaction.Data(string(d))
	return message, true
}
