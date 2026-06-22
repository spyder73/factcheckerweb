package auth

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

// Mailer abstracts transactional email so dev can run without SMTP creds.
// Phase 7 (deploy) wires SMTPMailer; Phase 1 ships LogMailer only.
type Mailer interface {
	SendVerifyEmail(ctx context.Context, to, link string) error
	SendPasswordReset(ctx context.Context, to, link string) error
}

// LogMailer prints email metadata to slog (dev only) and writes the raw link
// to a separate file at /tmp/alethea-dev-mail.log. This keeps tokens OUT of
// the main slog stream — which is typically shipped to centralized logging
// (Loki/Datadog/CloudWatch). A leaked log there would otherwise let an
// attacker take over any account that requested a reset.
//
// Dev workflow: tail /tmp/alethea-dev-mail.log in a second terminal.
type LogMailer struct{}

const devMailLogPath = "/tmp/alethea-dev-mail.log"

func (LogMailer) SendVerifyEmail(_ context.Context, to, link string) error {
	slog.Info("DEV email queued (verify)", "purpose", "verify_email", "note", "see "+devMailLogPath)
	return appendDevMailFile("verify", to, link)
}

func (LogMailer) SendPasswordReset(_ context.Context, to, link string) error {
	slog.Info("DEV email queued (reset)", "purpose", "reset_password", "note", "see "+devMailLogPath)
	return appendDevMailFile("reset", to, link)
}

// appendDevMailFile writes the raw token-bearing link to a local file. Best
// effort — failure is non-fatal, but logs a warning.
func appendDevMailFile(kind, to, link string) error {
	f, err := openMailLog()
	if err != nil {
		slog.Warn("LogMailer: could not open dev mail log; token not persisted to disk", "err", err, "to", to)
		return nil // signup/reset flow should still succeed in dev
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s\t%s\t%s\n", kind, to, link)
	return err
}

// openMailLog is split out so tests can stub it.
var openMailLog = func() (*os.File, error) {
	return os.OpenFile(devMailLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
}
