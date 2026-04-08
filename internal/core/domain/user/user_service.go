package user

import (
	"context"
	"strings"
	"time"
)

type Service struct {
	finder Finder
}

func NewService(finder Finder) *Service {
	return &Service{finder: finder}
}

func (s *Service) List(ctx context.Context, filter ListFilter) (*ListResult, error) {
	filter.Search = strings.TrimSpace(filter.Search)
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	totalCount, err := s.finder.Count(ctx, filter)
	if err != nil {
		return nil, err
	}

	users, err := s.finder.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &ListResult{
		TotalCount: totalCount,
		Users:      users,
	}, nil
}

func (s *Service) ListSessions(ctx context.Context, filter SessionListFilter) ([]LoginSession, error) {
	filter.UserID = strings.TrimSpace(filter.UserID)
	if filter.UserID == "" {
		return []LoginSession{}, nil
	}

	return s.finder.ListSessions(ctx, filter)
}

func (s *Service) RevokeAllSessions(ctx context.Context, userID string) (int64, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return 0, nil
	}
	return s.finder.RevokeSessionsByUserID(ctx, userID)
}

func (s *Service) RevokeSessionByID(ctx context.Context, userID, sessionID string) (bool, error) {
	userID = strings.TrimSpace(userID)
	sessionID = strings.TrimSpace(sessionID)
	if userID == "" || sessionID == "" {
		return false, nil
	}
	return s.finder.RevokeSessionByID(ctx, userID, sessionID)
}

func (s *Service) RevokeTokenFamily(ctx context.Context, userID, tokenFamilyID string) (int64, error) {
	userID = strings.TrimSpace(userID)
	tokenFamilyID = strings.TrimSpace(tokenFamilyID)
	if userID == "" || tokenFamilyID == "" {
		return 0, nil
	}
	return s.finder.RevokeSessionsByTokenFamily(ctx, userID, tokenFamilyID)
}

func (s *Service) CleanupInactiveSessions(ctx context.Context, retentionDays int) (int64, time.Time, error) {
	if retentionDays <= 0 {
		retentionDays = 90
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	deleted, err := s.finder.CleanupInactiveSessionsBefore(ctx, cutoff)
	if err != nil {
		return 0, cutoff, err
	}
	return deleted, cutoff, nil
}
