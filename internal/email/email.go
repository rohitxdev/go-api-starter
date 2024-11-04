// Package email provides utility functions for sending emails.
package email

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"text/template"

	"gopkg.in/gomail.v2"
)

type SMTPCredentials struct {
	Username string
	Password string
	Host     string
	Port     int
}

// Client is an email client that handles sending emails through SMTP.
type Client struct {
	dialer    *gomail.Dialer
	templates *template.Template
}

// New creates a new email client with the provided credentials and templates.
func New(c *SMTPCredentials, t *template.Template) *Client {
	return &Client{
		dialer:    gomail.NewDialer(c.Host, c.Port, c.Username, c.Password),
		templates: t,
	}
}

type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

type Headers struct {
	Subject     string
	FromAddress string
	FromName    string
	ToAddresses []string
	Cc          []string
	Bcc         []string
}

// SendRaw sends an email with raw content of the specified MIME type.
func (c *Client) SendRaw(headers *Headers, mimeType string, body string, attachments ...Attachment) error {
	msg := gomail.NewMessage()
	msg.SetHeaders(map[string][]string{
		"From":    {msg.FormatAddress(headers.FromAddress, headers.FromName)},
		"Subject": {headers.Subject},
		"To":      headers.ToAddresses,
	})

	if len(headers.Cc) > 0 {
		msg.SetHeader("Cc", headers.Cc...)
	}
	if len(headers.Bcc) > 0 {
		msg.SetHeader("Bcc", headers.Bcc...)
	}

	msg.SetBody(mimeType, body)

	for _, attachment := range attachments {
		if attachment.ContentType == "" {
			attachment.ContentType = http.DetectContentType(attachment.Data)
		}
		msg.Attach(
			attachment.Filename,
			gomail.SetCopyFunc(func(w io.Writer) error {
				_, err := w.Write(attachment.Data)
				return err
			}),
			gomail.SetHeader(map[string][]string{
				"Content-Type": {attachment.ContentType},
			}),
		)
	}

	if err := c.dialer.DialAndSend(msg); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	return nil
}

// SendHtml sends an HTML email using a template.
func (c *Client) SendHTML(headers *Headers, templateName string, data map[string]any, attachments ...Attachment) error {
	var buf bytes.Buffer
	if err := c.templates.ExecuteTemplate(&buf, templateName, data); err != nil {
		return fmt.Errorf("failed to execute template %q: %w", templateName, err)
	}
	return c.SendRaw(headers, "text/html", buf.String(), attachments...)
}

// SendText sends a plain text email.
func (c *Client) SendText(headers *Headers, body string, attachments ...Attachment) error {
	return c.SendRaw(headers, "text/plain", body, attachments...)
}
