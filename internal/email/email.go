// Package email provides utility functions for sending emails.
package email

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"text/template"

	"github.com/oklog/ulid/v2"
	"github.com/rohitxdev/go-api-starter/assets"
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
func New(c *SMTPCredentials) (*Client, error) {
	t, err := template.ParseFS(assets.FS, "templates/emails/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("Failed to parse email templates: %w", err)
	}
	client := Client{
		dialer:    gomail.NewDialer(c.Host, c.Port, c.Username, c.Password),
		templates: t,
	}
	return &client, nil
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
	// Prevent email stacking in the same thread on the email client.
	NoStack bool
}

// 'send' sends an email with raw content of the specified MIME type.
func (c *Client) send(headers *Headers, mimeType string, body string, attachments ...Attachment) error {
	msg := gomail.NewMessage()

	msg.SetHeaders(map[string][]string{
		"From":    {msg.FormatAddress(headers.FromAddress, headers.FromName)},
		"Subject": {headers.Subject},
		"To":      headers.ToAddresses,
	})

	if headers.NoStack {
		msg.SetHeader("X-Entity-Ref-ID", ulid.Make().String())
	}
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
		return fmt.Errorf("Failed to send email: %w", err)
	}
	return nil
}

// SendHtml sends an HTML email using a template.
func (c *Client) SendHTML(headers *Headers, templateName string, data map[string]any, attachments ...Attachment) error {
	var buf bytes.Buffer
	if err := c.templates.ExecuteTemplate(&buf, templateName, data); err != nil {
		// '%q' prints the template name in quotes
		return fmt.Errorf("Failed to execute template %q: %w", templateName, err)
	}
	return c.send(headers, "text/html", buf.String(), attachments...)
}

// SendText sends a plain text email.
func (c *Client) SendText(headers *Headers, body string, attachments ...Attachment) error {
	return c.send(headers, "text/plain", body, attachments...)
}
