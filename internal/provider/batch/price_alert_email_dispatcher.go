package batch

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	domainpricealert "github.com/ljj/gugu-admin-api/internal/core/domain/pricealert"
)

type PriceAlertEmailStore interface {
	ListDueEmailEvents(ctx context.Context, limit int) ([]domainpricealert.EmailNotificationEvent, error)
	TryClaimEmailEvent(ctx context.Context, event domainpricealert.EmailNotificationEvent) (bool, error)
	MarkEmailEventSent(ctx context.Context, event domainpricealert.EmailNotificationEvent) error
	MarkEmailEventFailed(ctx context.Context, event domainpricealert.EmailNotificationEvent, reason string) error
}

type PriceAlertEmailDispatcher struct {
	store       PriceAlertEmailStore
	sender      PriceAlertEmailSender
	statusStore *BatchStatusStore
	batchLimit  int
	runMu       sync.Mutex
}

func NewPriceAlertEmailDispatcher(
	store PriceAlertEmailStore,
	sender PriceAlertEmailSender,
	statusStore *BatchStatusStore,
	batchLimit int,
) *PriceAlertEmailDispatcher {
	if batchLimit <= 0 {
		batchLimit = 200
	}
	return &PriceAlertEmailDispatcher{
		store:       store,
		sender:      sender,
		statusStore: statusStore,
		batchLimit:  batchLimit,
	}
}

func (d *PriceAlertEmailDispatcher) CurrentStatus() (BatchJobStatus, bool) {
	if d == nil || d.statusStore == nil {
		return BatchJobStatus{}, false
	}
	return d.statusStore.Get(JobTypePriceAlertEmailScan)
}

func (d *PriceAlertEmailDispatcher) Queue(trigger TriggerType) BatchJobStatus {
	trigger = normalizeTriggerType(trigger)
	now := time.Now()
	status := BatchJobStatus{
		JobType:     JobTypePriceAlertEmailScan,
		Status:      JobStatusQueued,
		TriggerType: trigger,
		RequestedAt: now,
	}
	if d != nil && d.statusStore != nil {
		d.statusStore.Set(status)
	}
	return status
}

func (d *PriceAlertEmailDispatcher) Run(ctx context.Context, trigger TriggerType) (*PriceUpdateResult, error) {
	d.runMu.Lock()
	defer d.runMu.Unlock()

	if d == nil || d.store == nil {
		return nil, fmt.Errorf("price alert email store is not configured")
	}
	if d.sender == nil {
		return nil, fmt.Errorf("price alert email sender is not configured")
	}

	trigger = normalizeTriggerType(trigger)
	startedAt := time.Now()
	result := &PriceUpdateResult{
		JobType:    JobTypePriceAlertEmailScan,
		Status:     JobStatusRunning,
		StartedAt:  startedAt,
		TotalCount: 0,
	}

	d.updateStatus(trigger, result, nil)

	events, err := d.store.ListDueEmailEvents(ctx, d.batchLimit)
	if err != nil {
		result.Status = JobStatusFailed
		result.LastError = err.Error()
		finished := time.Now()
		result.FinishedAt = &finished
		d.updateStatus(trigger, result, &finished)
		return result, err
	}

	result.TotalCount = len(events)

	for _, event := range events {
		if ctx.Err() != nil {
			result.Status = JobStatusFailed
			result.LastError = ctx.Err().Error()
			break
		}

		claimed, claimErr := d.store.TryClaimEmailEvent(ctx, event)
		if claimErr != nil {
			result.FailCount++
			result.LastError = claimErr.Error()
			continue
		}
		if !claimed {
			result.SkippedCount++
			continue
		}

		sendErr := d.sender.SendPriceAlertEmail(ctx, event)
		if sendErr != nil {
			result.FailCount++
			result.LastError = sendErr.Error()
			_ = d.store.MarkEmailEventFailed(ctx, event, sendErr.Error())
			continue
		}

		if err := d.store.MarkEmailEventSent(ctx, event); err != nil {
			result.FailCount++
			result.LastError = err.Error()
			_ = d.store.MarkEmailEventFailed(ctx, event, "mark sent failed: "+err.Error())
			continue
		}

		result.SuccessCount++
	}

	finished := time.Now()
	result.FinishedAt = &finished
	if result.Status != JobStatusFailed {
		result.Status = JobStatusCompleted
	}

	d.updateStatus(trigger, result, &finished)
	return result, nil
}

func (d *PriceAlertEmailDispatcher) updateStatus(trigger TriggerType, result *PriceUpdateResult, finishedAt *time.Time) {
	if d == nil || d.statusStore == nil || result == nil {
		return
	}

	status := BatchJobStatus{
		JobType:      result.JobType,
		Status:       result.Status,
		TriggerType:  trigger,
		RequestedAt:  result.StartedAt,
		StartedAt:    &result.StartedAt,
		FinishedAt:   finishedAt,
		TotalCount:   result.TotalCount,
		SuccessCount: result.SuccessCount,
		FailCount:    result.FailCount,
		SkippedCount: result.SkippedCount,
		LastError:    strings.TrimSpace(result.LastError),
	}
	d.statusStore.Set(status)
}

func normalizeTriggerType(trigger TriggerType) TriggerType {
	if trigger == TriggerTypeScheduled {
		return TriggerTypeScheduled
	}
	return TriggerTypeManual
}
