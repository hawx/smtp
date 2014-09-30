package server

type Message struct {
	Sender     string
	Recipients []string
	Data       string
}

type Transaction interface {
	Sender(string)
	Recipient(string)
	Data(string)
	Message() Message
}

type transaction struct {
	sender     string
	recipients []string
	data       string
}

func NewTransaction() Transaction {
	return &transaction{}
}

func (t *transaction) Sender(sender string) {
	t.sender = sender
}

func (t *transaction) Recipient(recipient string) {
	t.recipients = append(t.recipients, recipient)
}

func (t *transaction) Data(data string) {
	t.data = data
}

func (t *transaction) Message() Message {
	return Message{t.sender, t.recipients, t.data}
}
