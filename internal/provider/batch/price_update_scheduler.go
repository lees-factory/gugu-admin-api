package batch

import (
	"context"
	"log"
	"time"
)

type PriceUpdateScheduler struct {
	updater  *PriceUpdater
	interval time.Duration
}

func NewPriceUpdateScheduler(updater *PriceUpdater, interval time.Duration) *PriceUpdateScheduler {
	return &PriceUpdateScheduler{
		updater:  updater,
		interval: interval,
	}
}

func (s *PriceUpdateScheduler) Start(ctx context.Context) {
	if s == nil || s.updater == nil || s.interval <= 0 {
		return
	}

	ticker := time.NewTicker(s.interval)

	go func() {
		defer ticker.Stop()

		log.Printf("price update scheduler started: interval=%s", s.interval)

		for {
			select {
			case <-ctx.Done():
				log.Printf("price update scheduler stopped: %v", ctx.Err())
				return
			case <-ticker.C:
				s.runOnce(ctx)
			}
		}
	}()
}

func (s *PriceUpdateScheduler) runOnce(ctx context.Context) {
	req := PriceUpdateRequest{
		TriggerType: TriggerTypeScheduled,
		RequestedBy: "internal-scheduler",
		Metadata: map[string]string{
			"runner": "internal-scheduler",
		},
	}

	status, err := s.updater.Preview(ctx, req)
	if err != nil {
		log.Printf("price update scheduler preview failed: %v", err)
		return
	}

	log.Printf("price update scheduler triggered: total=%d", status.TotalCount)

	if _, err := s.updater.Run(ctx, req); err != nil {
		log.Printf("price update scheduler run failed: %v", err)
	}
}
