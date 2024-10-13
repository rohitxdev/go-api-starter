// Package email provides utility functions for sending emails.
package email

import (
	"fmt"

	"gopkg.in/gomail.v2"
)

func NewSMTPClient(host string, port int, username string, password string) *gomail.Dialer {
	return gomail.NewDialer(host, port, username, password)
}

type Client struct {
	dialer *gomail.Dialer
}

/*----------------------------------- Send Email ----------------------------------- */

type Email struct {
	Subject     string
	ContentType string
	Body        string
	FromAddress string
	FromName    string
	ToAddresses []string
	Cc          []string
	Bcc         []string
}

func (c *Client) SendEmail(email *Email) error {
	msg := gomail.NewMessage()
	msg.SetHeaders(map[string][]string{
		"From":    {msg.FormatAddress(email.FromAddress, email.FromName)},
		"Subject": {email.Subject},
		"To":      email.ToAddresses,
		"Cc":      email.Cc,
		"Bcc":     email.Bcc,
	})
	msg.SetBody(email.ContentType, email.Body)
	if err := c.dialer.DialAndSend(msg); err != nil {
		return fmt.Errorf("could not send email: %w", err)
	}
	return nil
}
