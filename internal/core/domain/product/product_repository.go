package product

import (
	"context"

	"github.com/ljj/gugu-admin-api/internal/core/enum"
)

type Repository interface {
	FindByID(ctx context.Context, productID string) (*Product, error)
	FindByIDs(ctx context.Context, productIDs []string) ([]Product, error)
	FindByMarketAndExternalProductID(ctx context.Context, market enum.Market, externalProductID string) (*Product, error)
	ListByMarket(ctx context.Context, market enum.Market) ([]Product, error)
	ListByCollectionSource(ctx context.Context, collectionSource string) ([]Product, error)
	ListPriceUpdateCandidates(ctx context.Context, filter PriceUpdateCandidateFilter) ([]Product, error)
	ListAll(ctx context.Context) ([]Product, error)
	Create(ctx context.Context, product Product) error
	Update(ctx context.Context, product Product) error
}
