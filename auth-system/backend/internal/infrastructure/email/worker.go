// internal/infrastructure/email/worker.go
package email

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"tigersoft/auth-system/internal/config"
	"tigersoft/auth-system/internal/service"
)

// Worker consumes EmailTask values from a channel and sends them via the email Client.
// Implements the async email pattern (ADR-012).
type Worker struct {
	client *Client
	cfg    config.EmailConfig
}

// NewWorker creates a new email worker.
func NewWorker(client *Client, cfg config.EmailConfig) *Worker {
	return &Worker{client: client, cfg: cfg}
}

// Start launches cfg.WorkerConcurrency goroutines to consume from the channel.
// Runs until ctx is cancelled.
func (w *Worker) Start(ctx context.Context, ch <-chan service.EmailTask) {
	for i := 0; i < w.cfg.WorkerConcurrency; i++ {
		go w.consume(ctx, ch)
	}
	slog.Info("email worker goroutines started", "count", w.cfg.WorkerConcurrency)
}

// Drain waits for in-flight emails to complete (up to ctx deadline).
func (w *Worker) Drain(ctx context.Context) {
	// Simple drain: just wait for context.
	<-ctx.Done()
}

func (w *Worker) consume(ctx context.Context, ch <-chan service.EmailTask) {
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-ch:
			if !ok {
				return
			}
			w.send(ctx, task)
		}
	}
}

func (w *Worker) send(ctx context.Context, task service.EmailTask) {
	msg := w.buildMessage(task)

	var lastErr error
	backoff := w.cfg.RetryBackoffBase

	for attempt := 1; attempt <= w.cfg.MaxRetries; attempt++ {
		if err := w.client.Send(ctx, msg); err != nil {
			lastErr = err
			slog.Warn("email send failed, will retry",
				"type", task.Type,
				"to", task.ToEmail,
				"attempt", attempt,
				"error", err,
			)
			if attempt < w.cfg.MaxRetries {
				select {
				case <-ctx.Done():
					return
				case <-time.After(backoff):
				}
				backoff *= 2
			}
			continue
		}

		slog.Info("email sent", "type", task.Type, "to", task.ToEmail)
		return
	}

	slog.Error("email send failed after all retries",
		"type", task.Type,
		"to", task.ToEmail,
		"error", lastErr,
	)
}

func (w *Worker) buildMessage(task service.EmailTask) Message {
	switch task.Type {
	case service.EmailTypeVerification:
		return Message{
			To:       task.ToEmail,
			Subject:  "Verify your email address",
			HTMLBody: fmt.Sprintf(`<p>Hello %s,</p><p>Please verify your email address using the following token:</p><p><strong>%s</strong></p><p>This token expires at %s.</p>`, task.ToName, task.Token, task.ExpiresAt.Format(time.RFC3339)),
			TextBody: fmt.Sprintf("Hello %s,\n\nYour email verification token: %s\n\nExpires: %s", task.ToName, task.Token, task.ExpiresAt.Format(time.RFC3339)),
		}
	case service.EmailTypePasswordReset:
		return Message{
			To:       task.ToEmail,
			Subject:  "Reset your password",
			HTMLBody: fmt.Sprintf(`<p>Hello %s,</p><p>Use the following token to reset your password:</p><p><strong>%s</strong></p><p>This token expires at %s.</p>`, task.ToName, task.Token, task.ExpiresAt.Format(time.RFC3339)),
			TextBody: fmt.Sprintf("Hello %s,\n\nYour password reset token: %s\n\nExpires: %s", task.ToName, task.Token, task.ExpiresAt.Format(time.RFC3339)),
		}
	default:
		return Message{
			To:      task.ToEmail,
			Subject: "Notification",
			HTMLBody: fmt.Sprintf(`<p>Hello %s,</p><p>%s</p>`, task.ToName, task.Token),
		}
	}
}
