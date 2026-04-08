package batch

import (
	"context"
	"log"
	"time"

	domainuser "github.com/ljj/gugu-admin-api/internal/core/domain/user"
)

type SessionCleanupScheduler struct {
	userService   *domainuser.Service
	interval      time.Duration
	retentionDays int
}

func NewSessionCleanupScheduler(
	userService *domainuser.Service,
	interval time.Duration,
	retentionDays int,
) *SessionCleanupScheduler {
	return &SessionCleanupScheduler{
		userService:   userService,
		interval:      interval,
		retentionDays: retentionDays,
	}
}

func (s *SessionCleanupScheduler) Start(ctx context.Context) {
	if s == nil || s.userService == nil || s.interval <= 0 {
		return
	}

	startScheduleLoop(ctx, "session cleanup scheduler", s.interval, shouldAlignToMidnight(s.interval), s.runOnce)
}

func (s *SessionCleanupScheduler) runOnce(ctx context.Context) {
	deletedCount, cutoff, err := s.userService.CleanupInactiveSessions(ctx, s.retentionDays)
	if err != nil {
		log.Printf("session cleanup scheduler failed: retention_days=%d err=%v", s.retentionDays, err)
		return
	}

	log.Printf("session cleanup scheduler completed: retention_days=%d cutoff=%s deleted=%d",
		s.retentionDays, cutoff.Format(time.RFC3339), deletedCount)
}
