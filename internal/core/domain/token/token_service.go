package token

import (
	"context"
	"fmt"
	"time"
)

type IDGenerator interface {
	New() (string, error)
}

type Clock interface {
	Now() time.Time
}

type Service struct {
	repo  Repository
	idGen IDGenerator
	clock Clock
}

func NewService(repo Repository, idGen IDGenerator, clock Clock) *Service {
	return &Service{
		repo:  repo,
		idGen: idGen,
		clock: clock,
	}
}

func (s *Service) Now() time.Time {
	return s.clock.Now()
}

func (s *Service) GetByAppType(ctx context.Context, appType AppType) (*SellerToken, error) {
	return s.repo.GetByAppType(ctx, appType)
}

func (s *Service) GetAccessToken(ctx context.Context, appType AppType) (string, error) {
	t, err := s.repo.GetByAppType(ctx, appType)
	if err != nil {
		return "", fmt.Errorf("get token by app type %s: %w", appType, err)
	}
	if t == nil {
		return "", fmt.Errorf("no token found for app type %s", appType)
	}
	if t.IsAccessTokenExpired(s.clock.Now()) {
		return "", fmt.Errorf("access token expired for app type %s (expired at %s)", appType, t.AccessTokenExpiresAt)
	}
	return t.AccessToken, nil
}

func (s *Service) GetExpiringSoon(ctx context.Context, margin time.Duration) ([]SellerToken, error) {
	threshold := s.clock.Now().Add(margin)
	return s.repo.GetExpiringSoon(ctx, threshold)
}

func (s *Service) SaveToken(ctx context.Context, t SellerToken) error {
	if t.ID == "" {
		id, err := s.idGen.New()
		if err != nil {
			return fmt.Errorf("generate token id: %w", err)
		}
		t.ID = id
	}

	now := s.clock.Now()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now
	t.LastRefreshedAt = now

	return s.repo.Upsert(ctx, t)
}
