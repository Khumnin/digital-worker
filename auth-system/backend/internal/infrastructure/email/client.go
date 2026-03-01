// internal/infrastructure/email/client.go
package email

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"

	"github.com/resendlabs/resend-go/v2"
	"tigersoft/auth-system/internal/config"
)

// Client sends emails via Resend API (production) or SMTP/Mailhog (local dev).
type Client struct {
	resendClient *resend.Client
	cfg          config.EmailConfig
	useResend    bool
}

// NewClient creates an email client. Uses Resend if API key is set, SMTP otherwise.
func NewClient(cfg config.EmailConfig) *Client {
	if cfg.ResendAPIKey != "" {
		return &Client{
			resendClient: resend.NewClient(cfg.ResendAPIKey),
			cfg:          cfg,
			useResend:    true,
		}
	}
	slog.Info("RESEND_API_KEY not set — using SMTP/Mailhog for email delivery")
	return &Client{cfg: cfg, useResend: false}
}

// Send delivers an email message.
func (c *Client) Send(ctx context.Context, msg Message) error {
	if c.useResend {
		return c.sendViaResend(ctx, msg)
	}
	return c.sendViaSMTP(msg)
}

func (c *Client) sendViaResend(ctx context.Context, msg Message) error {
	params := &resend.SendEmailRequest{
		From:    fmt.Sprintf("%s <%s>", c.cfg.FromName, c.cfg.From),
		To:      []string{msg.To},
		Subject: msg.Subject,
		Html:    msg.HTMLBody,
	}
	if msg.TextBody != "" {
		params.Text = msg.TextBody
	}

	_, err := c.resendClient.Emails.SendWithContext(ctx, params)
	if err != nil {
		return fmt.Errorf("resend send email: %w", err)
	}
	return nil
}

func (c *Client) sendViaSMTP(msg Message) error {
	addr := fmt.Sprintf("%s:%d", c.cfg.SMTPHost, c.cfg.SMTPPort)

	body := fmt.Sprintf("From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		c.cfg.FromName, c.cfg.From, msg.To, msg.Subject, msg.HTMLBody)

	err := smtp.SendMail(addr, nil, c.cfg.From, []string{msg.To}, []byte(body))
	if err != nil {
		return fmt.Errorf("smtp send email: %w", err)
	}
	return nil
}

// Message represents an email to be sent.
type Message struct {
	To       string
	Subject  string
	HTMLBody string
	TextBody string
}
