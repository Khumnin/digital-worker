// internal/infrastructure/email/client.go
package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/smtp"

	"tigersoft/auth-system/internal/config"
)

// Client sends emails via Resend API (production) or SMTP/Mailhog (local dev).
type Client struct {
	httpClient *http.Client
	cfg        config.EmailConfig
	useResend  bool
}

// NewClient creates an email client. Uses Resend if API key is set, SMTP otherwise.
func NewClient(cfg config.EmailConfig) *Client {
	if cfg.ResendAPIKey != "" {
		return &Client{
			httpClient: &http.Client{},
			cfg:        cfg,
			useResend:  true,
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

// resendRequest mirrors the Resend v1 API body with tracking disabled.
type resendRequest struct {
	From        string `json:"from"`
	To          []string `json:"to"`
	Subject     string `json:"subject"`
	Html        string `json:"html,omitempty"`
	Text        string `json:"text,omitempty"`
	TrackClicks bool   `json:"track_clicks"`
	TrackOpens  bool   `json:"track_opens"`
}

func (c *Client) sendViaResend(ctx context.Context, msg Message) error {
	payload := resendRequest{
		From:        fmt.Sprintf("%s <%s>", c.cfg.FromName, c.cfg.From),
		To:          []string{msg.To},
		Subject:     msg.Subject,
		Html:        msg.HTMLBody,
		Text:        msg.TextBody,
		TrackClicks: false,
		TrackOpens:  false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal resend request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.ResendAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("resend send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend API error %d: %s", resp.StatusCode, string(respBody))
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
