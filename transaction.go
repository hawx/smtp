package server

type Message struct {
	Sender     string
	Recipients []string
	Data       string
}

type Transaction interface {
	Sender(string) (Transaction, bool)
	Recipient(string) (Transaction, bool)
	Data(string) (Message, bool)
}

func NewTransaction() Transaction {
	return &emptyTransaction{}
}

type emptyTransaction struct{}

func (t *emptyTransaction) Sender(sender string) (Transaction, bool) {
	return &senderTransaction{sender}, true
}

func (t *emptyTransaction) Recipient(recipient string) (Transaction, bool) {
	return nil, false
}

func (t *emptyTransaction) Data(data string) (Message, bool) {
	return Message{}, false
}


type senderTransaction struct {
	sender string
}

func (t *senderTransaction) Sender(sender string) (Transaction, bool) {
	return &senderTransaction{sender}, true
}

func (t *senderTransaction) Recipient(recipient string) (Transaction, bool) {
	return &recipientsTransaction{t.sender, []string{recipient}}, true
}

func (t *senderTransaction) Data(data string) (Message, bool) {
	return Message{}, false
}


type recipientsTransaction struct {
	sender     string
	recipients []string
}

func (t *recipientsTransaction) Sender(sender string) (Transaction, bool) {
	return &senderTransaction{sender}, true
}

func (t *recipientsTransaction) Recipient(recipient string) (Transaction, bool) {
	t.recipients = append(t.recipients, recipient)
	return t, true
}

func (t *recipientsTransaction) Data(data string) (Message, bool) {
	return Message{t.sender, t.recipients, data}, true
}
