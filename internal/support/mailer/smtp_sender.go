package mailer

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

// Config defines SMTP connection settings.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
}

// Sender sends a plain-text email message.
type Sender interface {
	Send(ctx context.Context, from, to, subject, body string) error
}

// SMTPSender sends email through a configured SMTP server.
type SMTPSender struct {
	host     string
	addr     string
	username string
	password string
}

func NewSMTPSender(cfg Config) (*SMTPSender, error) {
	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		return nil, fmt.Errorf("smtp host is required")
	}
	if cfg.Port <= 0 {
		return nil, fmt.Errorf("smtp port must be positive")
	}

	return &SMTPSender{
		host:     host,
		addr:     fmt.Sprintf("%s:%d", host, cfg.Port),
		username: strings.TrimSpace(cfg.Username),
		password: cfg.Password,
	}, nil
}

func (s *SMTPSender) Send(ctx context.Context, from, to, subject, body string) error {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	subject = strings.TrimSpace(subject)
	if from == "" {
		return fmt.Errorf("from address is required")
	}
	if to == "" {
		return fmt.Errorf("to address is required")
	}
	if subject == "" {
		subject = "Price alert"
	}

	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}

	header := []string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
	}
	message := strings.Join(header, "\r\n") + body

	errCh := make(chan error, 1)
	go func() {
		errCh <- smtp.SendMail(s.addr, auth, from, []string{to}, []byte(message))
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("smtp send failed: %w", err)
		}
		return nil
	}
}
