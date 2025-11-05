package email

// MockSender is a mock email sender for testing
type MockSender struct {
	SendFunc     func(to, subject, body string) error
	SendHTMLFunc func(to, subject, htmlBody, textBody string) error
	Calls        []SendCall
	HTMLCalls    []SendHTMLCall
}

// SendCall represents a call to Send
type SendCall struct {
	To      string
	Subject string
	Body    string
}

// SendHTMLCall represents a call to SendHTML
type SendHTMLCall struct {
	To       string
	Subject  string
	HTMLBody string
	TextBody string
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

// SendHTML records the call and optionally executes a custom function
func (m *MockSender) SendHTML(to, subject, htmlBody, textBody string) error {
	m.HTMLCalls = append(m.HTMLCalls, SendHTMLCall{
		To:       to,
		Subject:  subject,
		HTMLBody: htmlBody,
		TextBody: textBody,
	})

	if m.SendHTMLFunc != nil {
		return m.SendHTMLFunc(to, subject, htmlBody, textBody)
	}

	return nil
}
