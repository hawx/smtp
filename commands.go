package server

import (
	"regexp"
	"log"
)

var (
	mailRe = regexp.MustCompile("FROM:<(.*?)>")
	rcptRe = regexp.MustCompile("TO:<(.+)>")
)

func mail(args string, text connection, transaction Transaction) Transaction {
	matches := mailRe.FindStringSubmatch(args)
	if matches == nil || len(matches) != 2 {
		text.write(rSYNTAX_ERROR)
		return transaction
	}

	if newTransaction, ok := transaction.Sender(matches[1]); ok {
		text.write(rOK)
		return newTransaction
	}

	text.write(rOUT_OF_SEQUENCE)
	return transaction
}

func rcpt(args string, text connection, transaction Transaction) Transaction {
	matches := rcptRe.FindStringSubmatch(args)
	if matches == nil || len(matches) != 2 {
		text.write(rSYNTAX_ERROR)
		return transaction
	}

	if newTransaction, ok := transaction.Recipient(matches[1]); ok {
		text.write(rOK)
		return newTransaction
	} else {
		text.write(rOUT_OF_SEQUENCE)
		return transaction
	}
}

func data(text connection, transaction Transaction) (Message, bool) {
	if _, ok := transaction.Data("test"); !ok {
		text.write(rOUT_OF_SEQUENCE)
		return Message{}, false
	}

	text.write(rEND_DATA_WITH)

	data, err := text.readAll()
	if err != nil {
		log.Println("DATA:", err)
		return Message{}, false
	}

	text.write(rOK)
	message, _ := transaction.Data(data)
	return message, true
}
