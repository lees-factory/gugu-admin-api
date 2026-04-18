package adminauth

import (
	"context"
	"time"
)

type Repository interface {
	GetByID(ctx context.Context, id string) (*AdminUser, error)
	GetByLoginID(ctx context.Context, loginID string) (*AdminUser, error)
	UpdateLastLoginAt(ctx context.Context, id string, at time.Time) error
}
