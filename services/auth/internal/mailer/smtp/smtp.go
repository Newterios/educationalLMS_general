// Package smtp implements the Mailer port using Go's net/smtp package.
// Configure via environment variables:
//
//	SMTP_HOST        – e.g. smtp.gmail.com
//	SMTP_PORT        – e.g. 587
//	SMTP_USERNAME    – e.g. noreply@yourproject.com
//	SMTP_PASSWORD    – Gmail App Password (16 chars, no spaces)
//	SMTP_FROM        – display name + address, e.g. "EDULMS <noreply@...>"
//	SMTP_STARTTLS    – "true" to use STARTTLS (default), "false" for plain
//
// Usage:   smtp.New(smtp.Config{...})
package smtp

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"text/template"
	"time"
)

// Config holds SMTP connection settings.
type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	// StartTLS enables STARTTLS upgrade (required for Gmail :587).
	StartTLS bool
}

// ConfigFromEnv reads SMTP config from environment variables.
func ConfigFromEnv() Config {
	startTLS := os.Getenv("SMTP_STARTTLS") != "false" // default true
	return Config{
		Host:     os.Getenv("SMTP_HOST"),
		Port:     os.Getenv("SMTP_PORT"),
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     os.Getenv("SMTP_FROM"),
		StartTLS: startTLS,
	}
}

// Mailer sends transactional emails via SMTP.
type Mailer struct{ cfg Config }

// New creates a new SMTP Mailer. Call Ping() to validate connectivity.
func New(cfg Config) *Mailer { return &Mailer{cfg: cfg} }

// Ping dials the SMTP server to verify connectivity — useful in main().
func (m *Mailer) Ping() error {
	conn, err := net.DialTimeout("tcp", m.cfg.Host+":"+m.cfg.Port, 5*time.Second)
	if err != nil {
		return fmt.Errorf("smtp ping: %w", err)
	}
	conn.Close()
	return nil
}

// SendVerificationEmail sends an email-verification link.
func (m *Mailer) SendVerificationEmail(_ context.Context, to, fullName, code string) error {
	subject := "Verify your EDULMS account"
	body := renderVerification(fullName, code)
	return m.send(to, subject, body)
}

// SendPasswordResetEmail sends a one-time password-reset code.
func (m *Mailer) SendPasswordResetEmail(_ context.Context, to, fullName, code string) error {
	subject := "Reset your EDULMS password"
	body := renderReset(fullName, code)
	return m.send(to, subject, body)
}

// SendPasswordChangedEmail sends a confirmation that the password was changed.
func (m *Mailer) SendPasswordChangedEmail(_ context.Context, to, fullName string) error {
	subject := "Your EDULMS password was changed"
	body := fmt.Sprintf("Hi %s,\n\nYour password has been changed successfully.\n\nIf you did not request this, please contact support immediately.\n\nEDULMS Team", fullName)
	return m.send(to, subject, body)
}

// ── internals ──────────────────────────────────────────────────────────────

func (m *Mailer) send(to, subject, body string) error {
	addr := m.cfg.Host + ":" + m.cfg.Port
	auth := smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)

	msg := buildMessage(m.cfg.From, to, subject, body)

	if m.cfg.StartTLS {
		return sendStartTLS(addr, auth, m.cfg.Username, to, msg, m.cfg.Host)
	}
	return smtp.SendMail(addr, auth, m.cfg.Username, []string{to}, msg)
}

func sendStartTLS(addr string, auth smtp.Auth, from, to string, msg []byte, host string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer c.Quit() //nolint:errcheck

	tlsCfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	if err := c.StartTLS(tlsCfg); err != nil {
		return fmt.Errorf("smtp starttls: %w", err)
	}
	if err := c.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("smtp RCPT TO: %w", err)
	}
	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if _, err := wc.Write(msg); err != nil {
		return fmt.Errorf("smtp write body: %w", err)
	}
	return wc.Close()
}

func buildMessage(from, to, subject, body string) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "From: %s\r\n", from)
	fmt.Fprintf(&buf, "To: %s\r\n", to)
	fmt.Fprintf(&buf, "Subject: %s\r\n", subject)
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: text/plain; charset=UTF-8\r\n")
	fmt.Fprintf(&buf, "\r\n")
	fmt.Fprintf(&buf, "%s\r\n", body)
	return buf.Bytes()
}

var verificationTpl = template.Must(template.New("v").Parse(
	`Hi {{.Name}},

Welcome to EDULMS! Please verify your email address by entering this code:

    {{.Code}}

This code expires in 24 hours.

If you did not create an account, you can ignore this email.

EDULMS Team`))

func renderVerification(name, code string) string {
	var buf bytes.Buffer
	_ = verificationTpl.Execute(&buf, map[string]string{"Name": name, "Code": code})
	return buf.String()
}

var resetTpl = template.Must(template.New("r").Parse(
	`Hi {{.Name}},

You requested a password reset. Use this code:

    {{.Code}}

This code expires in 1 hour. If you did not request a reset, ignore this email.

EDULMS Team`))

func renderReset(name, code string) string {
	var buf bytes.Buffer
	_ = resetTpl.Execute(&buf, map[string]string{"Name": name, "Code": code})
	return buf.String()
}
