package port

import "context"

// Mailer sends transactional emails.
// Production impl uses Gmail SMTP; test impl writes to stdout.
type Mailer interface {
	SendVerificationEmail(ctx context.Context, to, fullName, code string) error
	SendPasswordResetEmail(ctx context.Context, to, fullName, code string) error
	SendPasswordChangedEmail(ctx context.Context, to, fullName string) error
}
