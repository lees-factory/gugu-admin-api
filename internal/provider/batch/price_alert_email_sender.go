package batch

import (
	"context"
	"fmt"
	"strings"

	domainpricealert "github.com/ljj/gugu-admin-api/internal/core/domain/pricealert"
	supportmailer "github.com/ljj/gugu-admin-api/internal/support/mailer"
)

type PriceAlertEmailSender interface {
	SendPriceAlertEmail(ctx context.Context, event domainpricealert.EmailNotificationEvent) error
}

type SMTPPriceAlertMailer struct {
	sender        supportmailer.Sender
	from          string
	subjectPrefix string
}

func NewSMTPPriceAlertMailer(sender supportmailer.Sender, from, subjectPrefix string) *SMTPPriceAlertMailer {
	return &SMTPPriceAlertMailer{
		sender:        sender,
		from:          strings.TrimSpace(from),
		subjectPrefix: strings.TrimSpace(subjectPrefix),
	}
}

func (m *SMTPPriceAlertMailer) SendPriceAlertEmail(ctx context.Context, event domainpricealert.EmailNotificationEvent) error {
	if m == nil || m.sender == nil {
		return fmt.Errorf("smtp price alert mailer is not configured")
	}

	subject := "Price Alert"
	if m.subjectPrefix != "" {
		subject = m.subjectPrefix + " " + subject
	}

	body := fmt.Sprintf(
		"Your tracked SKU price changed.\n\nSKU: %s\nCurrency: %s\nCurrent Price: %s\nChange Value: %s\nRecorded At: %s\n\nThis message was sent by GUGU price alert batch.",
		event.SKUID,
		event.Currency,
		event.Price,
		event.ChangeValue,
		event.RecordedAt.Format("2006-01-02T15:04:05Z07:00"),
	)

	return m.sender.Send(ctx, m.from, event.UserEmail, subject, body)
}
