package user

import (
	"context"
	"time"
)

type Finder interface {
	List(ctx context.Context, filter ListFilter) ([]User, error)
	Count(ctx context.Context, filter ListFilter) (int64, error)
	ListSessions(ctx context.Context, filter SessionListFilter) ([]LoginSession, error)
	RevokeSessionsByUserID(ctx context.Context, userID string) (int64, error)
	RevokeSessionByID(ctx context.Context, userID, sessionID string) (bool, error)
	RevokeSessionsByTokenFamily(ctx context.Context, userID, tokenFamilyID string) (int64, error)
	CleanupInactiveSessionsBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type finder struct {
	repository Repository
}

func NewFinder(repository Repository) Finder {
	return &finder{repository: repository}
}

func (f *finder) List(ctx context.Context, filter ListFilter) ([]User, error) {
	return f.repository.List(ctx, filter)
}

func (f *finder) Count(ctx context.Context, filter ListFilter) (int64, error) {
	return f.repository.Count(ctx, filter)
}

func (f *finder) ListSessions(ctx context.Context, filter SessionListFilter) ([]LoginSession, error) {
	return f.repository.ListSessions(ctx, filter)
}

func (f *finder) RevokeSessionsByUserID(ctx context.Context, userID string) (int64, error) {
	return f.repository.RevokeSessionsByUserID(ctx, userID)
}

func (f *finder) RevokeSessionByID(ctx context.Context, userID, sessionID string) (bool, error) {
	return f.repository.RevokeSessionByID(ctx, userID, sessionID)
}

func (f *finder) RevokeSessionsByTokenFamily(ctx context.Context, userID, tokenFamilyID string) (int64, error) {
	return f.repository.RevokeSessionsByTokenFamily(ctx, userID, tokenFamilyID)
}

func (f *finder) CleanupInactiveSessionsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	return f.repository.CleanupInactiveSessionsBefore(ctx, cutoff)
}
