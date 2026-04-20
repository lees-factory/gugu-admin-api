package pricealert

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	domainpricealert "github.com/ljj/gugu-admin-api/internal/core/domain/pricealert"
)

type Repository struct {
	db              *sql.DB
	claimRetryAfter time.Duration
}

func NewRepository(db *sql.DB, claimRetryAfter time.Duration) *Repository {
	if claimRetryAfter <= 0 {
		claimRetryAfter = 10 * time.Minute
	}
	return &Repository{db: db, claimRetryAfter: claimRetryAfter}
}

func (r *Repository) ListDueEmailEvents(ctx context.Context, limit int) ([]domainpricealert.EmailNotificationEvent, error) {
	if limit <= 0 {
		limit = 200
	}

	rows, err := r.db.QueryContext(ctx, listDueEmailEventsQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]domainpricealert.EmailNotificationEvent, 0, limit)
	for rows.Next() {
		var event domainpricealert.EmailNotificationEvent
		if err := rows.Scan(
			&event.AlertID,
			&event.UserID,
			&event.UserEmail,
			&event.SKUID,
			&event.Currency,
			&event.RecordedAt,
			&event.Price,
			&event.ChangeValue,
			&event.Channel,
		); err != nil {
			return nil, err
		}
		result = append(result, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Repository) TryClaimEmailEvent(ctx context.Context, event domainpricealert.EmailNotificationEvent) (bool, error) {
	channel := normalizeChannel(event.Channel)
	retryAfter := formatInterval(r.claimRetryAfter)

	var attemptCount int
	err := r.db.QueryRowContext(
		ctx,
		tryClaimEmailEventQuery,
		event.AlertID,
		event.SKUID,
		event.Currency,
		event.RecordedAt,
		channel,
		retryAfter,
	).Scan(&attemptCount)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (r *Repository) MarkEmailEventSent(ctx context.Context, event domainpricealert.EmailNotificationEvent) error {
	channel := normalizeChannel(event.Channel)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(
		ctx,
		markEmailEventSentQuery,
		event.AlertID,
		event.Currency,
		event.RecordedAt,
		channel,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("notification log not found for alert=%s sku=%s currency=%s recorded_at=%s",
			event.AlertID, event.SKUID, event.Currency, event.RecordedAt.Format(time.RFC3339))
	}

	if _, err := tx.ExecContext(
		ctx,
		markAlertCursorQuery,
		event.AlertID,
		event.RecordedAt,
		event.Currency,
	); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *Repository) MarkEmailEventFailed(ctx context.Context, event domainpricealert.EmailNotificationEvent, reason string) error {
	channel := normalizeChannel(event.Channel)
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "unknown send error"
	}

	res, err := r.db.ExecContext(
		ctx,
		markEmailEventFailedQuery,
		event.AlertID,
		event.Currency,
		event.RecordedAt,
		channel,
		reason,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("notification log not found for failed event: alert=%s currency=%s recorded_at=%s",
			event.AlertID, event.Currency, event.RecordedAt.Format(time.RFC3339))
	}

	return nil
}

func normalizeChannel(channel string) string {
	channel = strings.ToUpper(strings.TrimSpace(channel))
	if channel == "" {
		return "EMAIL"
	}
	return channel
}

func formatInterval(d time.Duration) string {
	if d <= 0 {
		d = 10 * time.Minute
	}
	seconds := int64(d / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	return fmt.Sprintf("%d seconds", seconds)
}

const listDueEmailEventsQuery = `
SELECT
    pa.id,
    pa.user_id,
    u.email,
    pa.sku_id,
    pa.currency,
    h_next.recorded_at,
    h_next.price,
    h_next.change_value,
    pa.channel
FROM gugu.price_alert pa
JOIN gugu.app_user u ON u.id = pa.user_id
JOIN LATERAL (
    SELECT
        h.recorded_at,
        h.price,
        h.change_value
    FROM gugu.sku_price_history h
    WHERE h.sku_id = pa.sku_id
      AND h.currency = pa.currency
      AND h.recorded_at > COALESCE(pa.last_notified_recorded_at, '0001-01-01 00:00:00+00'::timestamptz)
    ORDER BY h.recorded_at ASC
    LIMIT 1
) h_next ON TRUE
WHERE pa.enabled = TRUE
  AND UPPER(pa.channel) = 'EMAIL'
  AND u.email_verified = TRUE
ORDER BY h_next.recorded_at ASC, pa.id ASC
LIMIT $1;
`

const tryClaimEmailEventQuery = `
INSERT INTO gugu.price_alert_notification_log (
    alert_id,
    sku_id,
    currency,
    recorded_at,
    channel,
    status,
    attempt_count,
    claimed_at,
    created_at,
    updated_at
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    'PROCESSING',
    1,
    NOW(),
    NOW(),
    NOW()
)
ON CONFLICT (alert_id, currency, recorded_at, channel) DO UPDATE
SET status = 'PROCESSING',
    attempt_count = gugu.price_alert_notification_log.attempt_count + 1,
    claimed_at = NOW(),
    updated_at = NOW(),
    last_error = ''
WHERE gugu.price_alert_notification_log.status <> 'SENT'
  AND gugu.price_alert_notification_log.claimed_at <= NOW() - $6::interval
RETURNING attempt_count;
`

const markEmailEventSentQuery = `
UPDATE gugu.price_alert_notification_log
SET status = 'SENT',
    sent_at = NOW(),
    updated_at = NOW(),
    last_error = ''
WHERE alert_id = $1
  AND currency = $2
  AND recorded_at = $3
  AND channel = $4
  AND status <> 'SENT';
`

const markAlertCursorQuery = `
UPDATE gugu.price_alert
SET last_notified_recorded_at = GREATEST(
        COALESCE(last_notified_recorded_at, '0001-01-01 00:00:00+00'::timestamptz),
        $2
    ),
    last_notified_currency = $3,
    last_notified_at = NOW()
WHERE id = $1;
`

const markEmailEventFailedQuery = `
UPDATE gugu.price_alert_notification_log
SET status = 'FAILED',
    updated_at = NOW(),
    last_error = $5
WHERE alert_id = $1
  AND currency = $2
  AND recorded_at = $3
  AND channel = $4
  AND status <> 'SENT';
`
