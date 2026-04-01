package token

import (
	"context"
	"time"
)

type Repository interface {
	GetByAppType(ctx context.Context, appType AppType) (*SellerToken, error)
	GetBySellerID(ctx context.Context, sellerID string) (*SellerToken, error)
	GetExpiringSoon(ctx context.Context, threshold time.Time) ([]SellerToken, error)
	Upsert(ctx context.Context, token SellerToken) error
}
