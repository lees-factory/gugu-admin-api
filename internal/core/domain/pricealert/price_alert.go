package pricealert

import "time"

// EmailNotificationEvent represents one price-change event for one alert recipient.
type EmailNotificationEvent struct {
	AlertID     string
	UserID      string
	UserEmail   string
	SKUID       string
	Currency    string
	RecordedAt  time.Time
	Price       string
	ChangeValue string
	Channel     string
}
