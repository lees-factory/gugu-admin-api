package user

import (
	"context"
	"time"
)

type Repository interface {
	List(ctx context.Context, filter ListFilter) ([]User, error)
	Count(ctx context.Context, filter ListFilter) (int64, error)
	ListSessions(ctx context.Context, filter SessionListFilter) ([]LoginSession, error)
	RevokeSessionsByUserID(ctx context.Context, userID string) (int64, error)
	RevokeSessionByID(ctx context.Context, userID, sessionID string) (bool, error)
	RevokeSessionsByTokenFamily(ctx context.Context, userID, tokenFamilyID string) (int64, error)
	CleanupInactiveSessionsBefore(ctx context.Context, cutoff time.Time) (int64, error)
}
