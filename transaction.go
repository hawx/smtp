package smtp

type Message struct {
	Sender     string
	Recipients []string
	Data       string
}

type transaction interface {
	Sender(string) (transaction, bool)
	Recipient(string) (transaction, bool)
	Data(string) (Message, bool)
}

func newTransaction() transaction {
	return &closedTransaction{}
}

func resetTransaction(t transaction) transaction {
	return &emptyTransaction{}
}

type closedTransaction struct{}

func (t *closedTransaction) Sender(sender string) (transaction, bool) {
	return nil, false
}

func (t *closedTransaction) Recipient(recipient string) (transaction, bool) {
	return nil, false
}

func (t *closedTransaction) Data(data string) (Message, bool) {
	return Message{}, false
}

type emptyTransaction struct{}

func (t *emptyTransaction) Sender(sender string) (transaction, bool) {
	return &senderTransaction{sender}, true
}

func (t *emptyTransaction) Recipient(recipient string) (transaction, bool) {
	return nil, false
}

func (t *emptyTransaction) Data(data string) (Message, bool) {
	return Message{}, false
}


type senderTransaction struct {
	sender string
}

func (t *senderTransaction) Sender(sender string) (transaction, bool) {
	return &senderTransaction{sender}, true
}

func (t *senderTransaction) Recipient(recipient string) (transaction, bool) {
	return &recipientsTransaction{t.sender, []string{recipient}}, true
}

func (t *senderTransaction) Data(data string) (Message, bool) {
	return Message{}, false
}


type recipientsTransaction struct {
	sender     string
	recipients []string
}

func (t *recipientsTransaction) Sender(sender string) (transaction, bool) {
	return &senderTransaction{sender}, true
}

func (t *recipientsTransaction) Recipient(recipient string) (transaction, bool) {
	t.recipients = append(t.recipients, recipient)
	return t, true
}

func (t *recipientsTransaction) Data(data string) (Message, bool) {
	return Message{t.sender, t.recipients, data}, true
}
