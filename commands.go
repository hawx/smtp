package smtp

import (
	"regexp"
	"log"
)

var (
	mailRe = regexp.MustCompile("FROM:<(.*?)>")
	rcptRe = regexp.MustCompile("TO:<(.+)>")
)

func mail(args string, text connection, tran transaction) transaction {
	matches := mailRe.FindStringSubmatch(args)
	if matches == nil || len(matches) != 2 {
		text.write(rSYNTAX_ERROR)
		return tran
	}

	if newTransaction, ok := tran.Sender(matches[1]); ok {
		text.write(rOK)
		return newTransaction
	}

	text.write(rOUT_OF_SEQUENCE)
	return tran
}

func rcpt(args string, text connection, tran transaction) transaction {
	matches := rcptRe.FindStringSubmatch(args)
	if matches == nil || len(matches) != 2 {
		text.write(rSYNTAX_ERROR)
		return tran
	}

	if newTransaction, ok := tran.Recipient(matches[1]); ok {
		text.write(rOK)
		return newTransaction
	} else {
		text.write(rOUT_OF_SEQUENCE)
		return tran
	}
}

func data(text connection, tran transaction) (Message, bool) {
	if _, ok := tran.Data([]byte{}); !ok {
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
	message, _ := tran.Data(data)
	return message, true
}
