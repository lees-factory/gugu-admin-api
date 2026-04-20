package batch

import (
	"context"
	"log"
	"time"
)

type PriceAlertEmailScheduler struct {
	dispatcher *PriceAlertEmailDispatcher
	interval   time.Duration
}

func NewPriceAlertEmailScheduler(dispatcher *PriceAlertEmailDispatcher, interval time.Duration) *PriceAlertEmailScheduler {
	return &PriceAlertEmailScheduler{
		dispatcher: dispatcher,
		interval:   interval,
	}
}

func (s *PriceAlertEmailScheduler) Start(ctx context.Context) {
	if s == nil || s.dispatcher == nil || s.interval <= 0 {
		return
	}

	startScheduleLoop(ctx, "price alert email scheduler", s.interval, shouldAlignToMidnight(s.interval), s.runOnce)
}

func (s *PriceAlertEmailScheduler) runOnce(ctx context.Context) {
	result, err := s.dispatcher.Run(ctx, TriggerTypeScheduled)
	if err != nil {
		log.Printf("price alert email scheduler failed: %v", err)
		return
	}
	log.Printf("price alert email scheduler completed: total=%d success=%d fail=%d skipped=%d",
		result.TotalCount, result.SuccessCount, result.FailCount, result.SkippedCount)
}
