package email

// MockSender is a mock email sender for testing
type MockSender struct {
	SendFunc func(to, subject, body string) error
	Calls    []SendCall
}

// SendCall represents a call to Send
type SendCall struct {
	To      string
	Subject string
	Body    string
}

// Send records the call and optionally executes a custom function
func (m *MockSender) Send(to, subject, body string) error {
	m.Calls = append(m.Calls, SendCall{
		To:      to,
		Subject: subject,
		Body:    body,
	})

	if m.SendFunc != nil {
		return m.SendFunc(to, subject, body)
	}

	return nil
}
