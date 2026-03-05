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
	case service.EmailTypeInvitation:
		return w.buildInvitationMessage(task)
	case service.EmailTypeVerification:
		return w.buildVerificationMessage(task)
	case service.EmailTypePasswordReset:
		return w.buildPasswordResetMessage(task)
	default:
		return Message{
			To:       task.ToEmail,
			Subject:  "Notification",
			HTMLBody: fmt.Sprintf(`<p>Hello %s,</p><p>%s</p>`, task.ToName, task.Token),
		}
	}
}

func (w *Worker) buildInvitationMessage(task service.EmailTask) Message {
	expiryDate := task.ExpiresAt.Format("2 January 2006")
	acceptURL := fmt.Sprintf("%s/accept-invite?token=%s", w.cfg.AppURL, task.Token)
	if task.TenantSlug != "" {
		acceptURL += "&tenant=" + task.TenantSlug
	}

	// Use the tenant's display name; fall back to "TigerSoft" when absent.
	tenantName := task.TenantName
	if tenantName == "" {
		tenantName = "TigerSoft"
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;padding:0;background:#f5f5f5;font-family:'Helvetica Neue',Helvetica,Arial,sans-serif;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background:#f5f5f5;padding:40px 0;">
    <tr><td align="center">
      <table width="560" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:10px;overflow:hidden;box-shadow:0 1px 4px rgba(0,0,0,0.08);">

        <!-- Header -->
        <tr>
          <td style="background:#C10016;padding:28px 40px;">
            <span style="color:#ffffff;font-size:20px;font-weight:700;letter-spacing:-0.3px;">%s</span>
          </td>
        </tr>

        <!-- Body -->
        <tr>
          <td style="padding:40px 40px 32px;">
            <h1 style="margin:0 0 16px;font-size:22px;font-weight:700;color:#1a1a1a;">You've been invited</h1>
            <p style="margin:0 0 12px;font-size:15px;color:#444;line-height:1.6;">
              Hello %s,
            </p>
            <p style="margin:0 0 28px;font-size:15px;color:#444;line-height:1.6;">
              You have been invited to join <strong>%s</strong>. Click the button below to set up your account and get started.
            </p>

            <!-- CTA Button -->
            <table cellpadding="0" cellspacing="0" style="margin-bottom:28px;">
              <tr>
                <td style="background:#C10016;border-radius:1000px;padding:14px 32px;">
                  <a href="%s" style="color:#ffffff;font-size:15px;font-weight:600;text-decoration:none;display:inline-block;">
                    Accept Invitation →
                  </a>
                </td>
              </tr>
            </table>

            <p style="margin:0 0 8px;font-size:13px;color:#888;line-height:1.6;">
              If the button doesn't work, copy and paste this link into your browser:
            </p>
            <p style="margin:0 0 28px;font-size:13px;color:#C10016;word-break:break-all;">
              <a href="%s" style="color:#C10016;">%s</a>
            </p>

            <p style="margin:0;font-size:13px;color:#aaa;">
              This invitation expires on <strong>%s</strong>. If you did not expect this invitation, you can safely ignore this email.
            </p>
          </td>
        </tr>

        <!-- Footer -->
        <tr>
          <td style="padding:20px 40px;border-top:1px solid #f0f0f0;">
            <p style="margin:0;font-size:12px;color:#bbb;text-align:center;">
              © %s · This is an automated message, please do not reply.
            </p>
          </td>
        </tr>

      </table>
    </td></tr>
  </table>
</body>
</html>`, tenantName, task.ToName, tenantName, acceptURL, acceptURL, acceptURL, expiryDate, tenantName)

	text := fmt.Sprintf(
		"Hello %s,\n\nYou have been invited to join %s.\n\nAccept your invitation here:\n%s\n\nThis invitation expires on %s.\n\nIf you did not expect this invitation, please ignore this email.",
		task.ToName, tenantName, acceptURL, expiryDate,
	)

	return Message{
		To:       task.ToEmail,
		Subject:  fmt.Sprintf("You've been invited to %s", tenantName),
		HTMLBody: html,
		TextBody: text,
	}
}

func (w *Worker) buildVerificationMessage(task service.EmailTask) Message {
	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", w.cfg.AppURL, task.Token)
	expiryDate := task.ExpiresAt.Format("2 January 2006 15:04 MST")

	tenantName := task.TenantName
	if tenantName == "" {
		tenantName = "TigerSoft"
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"></head>
<body style="margin:0;padding:0;background:#f5f5f5;font-family:'Helvetica Neue',Helvetica,Arial,sans-serif;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background:#f5f5f5;padding:40px 0;">
    <tr><td align="center">
      <table width="560" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:10px;overflow:hidden;box-shadow:0 1px 4px rgba(0,0,0,0.08);">
        <tr><td style="background:#C10016;padding:28px 40px;">
          <span style="color:#ffffff;font-size:20px;font-weight:700;">%s</span>
        </td></tr>
        <tr><td style="padding:40px;">
          <h1 style="margin:0 0 16px;font-size:22px;font-weight:700;color:#1a1a1a;">Verify your email address</h1>
          <p style="margin:0 0 28px;font-size:15px;color:#444;line-height:1.6;">Hello %s, please verify your email address by clicking the button below.</p>
          <table cellpadding="0" cellspacing="0" style="margin-bottom:28px;">
            <tr><td style="background:#C10016;border-radius:1000px;padding:14px 32px;">
              <a href="%s" style="color:#ffffff;font-size:15px;font-weight:600;text-decoration:none;">Verify Email →</a>
            </td></tr>
          </table>
          <p style="margin:0;font-size:13px;color:#aaa;">Expires: %s</p>
        </td></tr>
        <tr><td style="padding:20px 40px;border-top:1px solid #f0f0f0;">
          <p style="margin:0;font-size:12px;color:#bbb;text-align:center;">© %s · Automated message, do not reply.</p>
        </td></tr>
      </table>
    </td></tr>
  </table>
</body></html>`, tenantName, task.ToName, verifyURL, expiryDate, tenantName)

	return Message{
		To:       task.ToEmail,
		Subject:  "Verify your email address",
		HTMLBody: html,
		TextBody: fmt.Sprintf("Hello %s,\n\nVerify your email: %s\n\nExpires: %s", task.ToName, verifyURL, expiryDate),
	}
}

func (w *Worker) buildPasswordResetMessage(task service.EmailTask) Message {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", w.cfg.AppURL, task.Token)
	expiryDate := task.ExpiresAt.Format("2 January 2006 15:04 MST")

	tenantName := task.TenantName
	if tenantName == "" {
		tenantName = "TigerSoft"
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"></head>
<body style="margin:0;padding:0;background:#f5f5f5;font-family:'Helvetica Neue',Helvetica,Arial,sans-serif;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background:#f5f5f5;padding:40px 0;">
    <tr><td align="center">
      <table width="560" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:10px;overflow:hidden;box-shadow:0 1px 4px rgba(0,0,0,0.08);">
        <tr><td style="background:#C10016;padding:28px 40px;">
          <span style="color:#ffffff;font-size:20px;font-weight:700;">%s</span>
        </td></tr>
        <tr><td style="padding:40px;">
          <h1 style="margin:0 0 16px;font-size:22px;font-weight:700;color:#1a1a1a;">Reset your password</h1>
          <p style="margin:0 0 28px;font-size:15px;color:#444;line-height:1.6;">Hello %s, click the button below to reset your password. This link is valid for 1 hour.</p>
          <table cellpadding="0" cellspacing="0" style="margin-bottom:28px;">
            <tr><td style="background:#C10016;border-radius:1000px;padding:14px 32px;">
              <a href="%s" style="color:#ffffff;font-size:15px;font-weight:600;text-decoration:none;">Reset Password →</a>
            </td></tr>
          </table>
          <p style="margin:0;font-size:13px;color:#aaa;">If you did not request a password reset, ignore this email. Expires: %s</p>
        </td></tr>
        <tr><td style="padding:20px 40px;border-top:1px solid #f0f0f0;">
          <p style="margin:0;font-size:12px;color:#bbb;text-align:center;">© %s · Automated message, do not reply.</p>
        </td></tr>
      </table>
    </td></tr>
  </table>
</body></html>`, tenantName, task.ToName, resetURL, expiryDate, tenantName)

	return Message{
		To:       task.ToEmail,
		Subject:  "Reset your password",
		HTMLBody: html,
		TextBody: fmt.Sprintf("Hello %s,\n\nReset your password: %s\n\nExpires: %s\n\nIf you did not request this, ignore this email.", task.ToName, resetURL, expiryDate),
	}
}
