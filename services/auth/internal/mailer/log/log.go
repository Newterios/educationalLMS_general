package log

import (
	"context"
	"log/slog"
)

// Mailer is a log-only Mailer that writes outgoing mails to stdout.
// Used in Phase 1C. Real SMTP + mock-gateway replaces it in Phase 2.
type Mailer struct{}

func New() *Mailer { return &Mailer{} }

func (m *Mailer) SendVerificationEmail(_ context.Context, to, fullName, code string) error {
	slog.Info("would send verification email", "to", to, "name", fullName, "code", code)
	return nil
}

func (m *Mailer) SendPasswordResetEmail(_ context.Context, to, fullName, code string) error {
	slog.Info("would send password reset email", "to", to, "name", fullName, "code", code)
	return nil
}

func (m *Mailer) SendPasswordChangedEmail(_ context.Context, to, fullName string) error {
	slog.Info("would send password changed email", "to", to, "name", fullName)
	return nil
}
