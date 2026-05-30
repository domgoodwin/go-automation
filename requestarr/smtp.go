package requestarr

import (
	"crypto/tls"
	"fmt"
	"path/filepath"
	"strconv"

	gomail "gopkg.in/mail.v2"
)

type SMTPSender struct {
	host     string
	port     int
	username string
	password string
}

func NewSMTPSender(host, portStr, username, password string) (*SMTPSender, error) {
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP port %q: %w", portStr, err)
	}
	return &SMTPSender{host: host, port: port, username: username, password: password}, nil
}

// SendBook sends a book file as an email attachment to a Kindle email address.
// Using subject "convert" tells Amazon to convert the format if needed.
// The sender address must be on the recipient's Amazon approved sender list.
func (s *SMTPSender) SendBook(to, bookFile, bookTitle, author string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.username)
	m.SetHeader("To", to)
	m.SetHeader("Subject", "convert")
	body := fmt.Sprintf("Sending: %s", bookTitle)
	if author != "" {
		body += " by " + author
	}
	m.SetBody("text/plain", body)
	m.Attach(bookFile, gomail.Rename(filepath.Base(bookFile)))

	d := gomail.NewDialer(s.host, s.port, s.username, s.password)
	d.TLSConfig = &tls.Config{ServerName: s.host}
	return d.DialAndSend(m)
}
