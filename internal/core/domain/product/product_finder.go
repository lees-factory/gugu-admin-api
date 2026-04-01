package product

import (
	"context"

	"github.com/ljj/gugu-admin-api/internal/core/enum"
)

type Finder interface {
	FindByID(ctx context.Context, productID string) (*Product, error)
	FindByIDs(ctx context.Context, productIDs []string) ([]Product, error)
	FindByMarketAndExternalProductID(ctx context.Context, market enum.Market, externalProductID string) (*Product, error)
	ListByMarket(ctx context.Context, market enum.Market) ([]Product, error)
	ListByCollectionSource(ctx context.Context, collectionSource string) ([]Product, error)
	ListPriceUpdateCandidates(ctx context.Context, filter PriceUpdateCandidateFilter) ([]Product, error)
	ListAll(ctx context.Context) ([]Product, error)
}

type finder struct {
	repository Repository
}

func NewFinder(repository Repository) Finder {
	return &finder{repository: repository}
}

func (f *finder) FindByID(ctx context.Context, productID string) (*Product, error) {
	return f.repository.FindByID(ctx, productID)
}

func (f *finder) FindByIDs(ctx context.Context, productIDs []string) ([]Product, error) {
	return f.repository.FindByIDs(ctx, productIDs)
}

func (f *finder) FindByMarketAndExternalProductID(ctx context.Context, market enum.Market, externalProductID string) (*Product, error) {
	return f.repository.FindByMarketAndExternalProductID(ctx, market, externalProductID)
}

func (f *finder) ListByMarket(ctx context.Context, market enum.Market) ([]Product, error) {
	return f.repository.ListByMarket(ctx, market)
}

func (f *finder) ListByCollectionSource(ctx context.Context, collectionSource string) ([]Product, error) {
	return f.repository.ListByCollectionSource(ctx, collectionSource)
}

func (f *finder) ListPriceUpdateCandidates(ctx context.Context, filter PriceUpdateCandidateFilter) ([]Product, error) {
	return f.repository.ListPriceUpdateCandidates(ctx, filter)
}

func (f *finder) ListAll(ctx context.Context) ([]Product, error) {
	return f.repository.ListAll(ctx)
}
