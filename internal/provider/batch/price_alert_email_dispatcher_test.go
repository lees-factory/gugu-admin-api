package batch

import (
	"context"
	"errors"
	"testing"
	"time"

	domainpricealert "github.com/ljj/gugu-admin-api/internal/core/domain/pricealert"
)

type fakePriceAlertEmailStore struct {
	events     []domainpricealert.EmailNotificationEvent
	claimErr   error
	sendMarked []domainpricealert.EmailNotificationEvent
	failMarked []domainpricealert.EmailNotificationEvent

	claimable map[string]bool
}

func (f *fakePriceAlertEmailStore) ListDueEmailEvents(ctx context.Context, limit int) ([]domainpricealert.EmailNotificationEvent, error) {
	if limit > 0 && len(f.events) > limit {
		return append([]domainpricealert.EmailNotificationEvent{}, f.events[:limit]...), nil
	}
	return append([]domainpricealert.EmailNotificationEvent{}, f.events...), nil
}

func (f *fakePriceAlertEmailStore) TryClaimEmailEvent(ctx context.Context, event domainpricealert.EmailNotificationEvent) (bool, error) {
	if f.claimErr != nil {
		return false, f.claimErr
	}
	if f.claimable == nil {
		return true, nil
	}
	allowed, ok := f.claimable[event.AlertID]
	if !ok {
		return true, nil
	}
	return allowed, nil
}

func (f *fakePriceAlertEmailStore) MarkEmailEventSent(ctx context.Context, event domainpricealert.EmailNotificationEvent) error {
	f.sendMarked = append(f.sendMarked, event)
	return nil
}

func (f *fakePriceAlertEmailStore) MarkEmailEventFailed(ctx context.Context, event domainpricealert.EmailNotificationEvent, reason string) error {
	f.failMarked = append(f.failMarked, event)
	return nil
}

type fakePriceAlertEmailSender struct {
	failFor map[string]error
	sent    []domainpricealert.EmailNotificationEvent
}

func (f *fakePriceAlertEmailSender) SendPriceAlertEmail(ctx context.Context, event domainpricealert.EmailNotificationEvent) error {
	if f.failFor != nil {
		if err, ok := f.failFor[event.AlertID]; ok {
			return err
		}
	}
	f.sent = append(f.sent, event)
	return nil
}

func TestPriceAlertEmailDispatcher_Run(t *testing.T) {
	now := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	events := []domainpricealert.EmailNotificationEvent{
		{AlertID: "a1", SKUID: "s1", Currency: "KRW", RecordedAt: now, UserEmail: "a@example.com", Channel: "EMAIL"},
		{AlertID: "a2", SKUID: "s2", Currency: "USD", RecordedAt: now.Add(time.Minute), UserEmail: "b@example.com", Channel: "EMAIL"},
		{AlertID: "a3", SKUID: "s3", Currency: "USD", RecordedAt: now.Add(2 * time.Minute), UserEmail: "c@example.com", Channel: "EMAIL"},
	}

	store := &fakePriceAlertEmailStore{
		events: events,
		claimable: map[string]bool{
			"a3": false,
		},
	}
	sender := &fakePriceAlertEmailSender{
		failFor: map[string]error{
			"a2": errors.New("smtp down"),
		},
	}
	statusStore := NewBatchStatusStore()
	dispatcher := NewPriceAlertEmailDispatcher(store, sender, statusStore, 100)

	result, err := dispatcher.Run(context.Background(), TriggerTypeManual)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.TotalCount != 3 {
		t.Fatalf("TotalCount = %d, want 3", result.TotalCount)
	}
	if result.SuccessCount != 1 {
		t.Fatalf("SuccessCount = %d, want 1", result.SuccessCount)
	}
	if result.FailCount != 1 {
		t.Fatalf("FailCount = %d, want 1", result.FailCount)
	}
	if result.SkippedCount != 1 {
		t.Fatalf("SkippedCount = %d, want 1", result.SkippedCount)
	}
	if len(store.sendMarked) != 1 {
		t.Fatalf("marked sent count = %d, want 1", len(store.sendMarked))
	}
	if len(store.failMarked) != 1 {
		t.Fatalf("marked failed count = %d, want 1", len(store.failMarked))
	}

	status, ok := dispatcher.CurrentStatus()
	if !ok {
		t.Fatal("CurrentStatus() not found")
	}
	if status.JobType != JobTypePriceAlertEmailScan {
		t.Fatalf("job type = %s, want %s", status.JobType, JobTypePriceAlertEmailScan)
	}
	if status.Status != JobStatusCompleted {
		t.Fatalf("status = %s, want %s", status.Status, JobStatusCompleted)
	}
}
