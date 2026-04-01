package product

import (
	"context"
	"database/sql"
	"time"

	domainproduct "github.com/ljj/gugu-admin-api/internal/core/domain/product"
	"github.com/ljj/gugu-admin-api/internal/core/enum"
	"github.com/ljj/gugu-admin-api/internal/storage/dbcore/sqldb"
)

type SQLCRepository struct {
	queries *sqldb.Queries
}

func NewSQLCRepository(db *sql.DB) *SQLCRepository {
	return &SQLCRepository{queries: sqldb.New(db)}
}

func (r *SQLCRepository) FindByID(ctx context.Context, productID string) (*domainproduct.Product, error) {
	row, err := r.queries.FindProductByID(ctx, productID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	p := toDomainProduct(row)
	return &p, nil
}

func (r *SQLCRepository) FindByIDs(ctx context.Context, productIDs []string) ([]domainproduct.Product, error) {
	rows, err := r.queries.FindProductsByIDs(ctx, productIDs)
	if err != nil {
		return nil, err
	}
	result := make([]domainproduct.Product, len(rows))
	for i, row := range rows {
		result[i] = toDomainProduct(row)
	}
	return result, nil
}

func (r *SQLCRepository) FindByMarketAndExternalProductID(ctx context.Context, market enum.Market, externalProductID string) (*domainproduct.Product, error) {
	row, err := r.queries.FindProductByMarketAndExternalProductID(ctx, sqldb.FindProductByMarketAndExternalProductIDParams{
		Market:            string(market),
		ExternalProductID: externalProductID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	p := toDomainProduct(row)
	return &p, nil
}

func (r *SQLCRepository) ListByMarket(ctx context.Context, market enum.Market) ([]domainproduct.Product, error) {
	rows, err := r.queries.ListProductsByMarket(ctx, string(market))
	if err != nil {
		return nil, err
	}
	result := make([]domainproduct.Product, len(rows))
	for i, row := range rows {
		result[i] = toDomainProduct(row)
	}
	return result, nil
}

func (r *SQLCRepository) ListByCollectionSource(ctx context.Context, collectionSource string) ([]domainproduct.Product, error) {
	rows, err := r.queries.ListProductsByCollectionSource(ctx, collectionSource)
	if err != nil {
		return nil, err
	}
	result := make([]domainproduct.Product, len(rows))
	for i, row := range rows {
		result[i] = toDomainProduct(row)
	}
	return result, nil
}

func (r *SQLCRepository) ListPriceUpdateCandidates(ctx context.Context, filter domainproduct.PriceUpdateCandidateFilter) ([]domainproduct.Product, error) {
	collectedBefore := time.Time{}
	if filter.CollectedBefore != nil {
		collectedBefore = *filter.CollectedBefore
	}

	rows, err := r.queries.ListPriceUpdateCandidateProducts(ctx, sqldb.ListPriceUpdateCandidateProductsParams{
		Column1: filter.CollectionSource,
		Column2: string(filter.Market),
		Column3: collectedBefore,
	})
	if err != nil {
		return nil, err
	}
	result := make([]domainproduct.Product, len(rows))
	for i, row := range rows {
		result[i] = toDomainProduct(row)
	}
	return result, nil
}

func (r *SQLCRepository) ListAll(ctx context.Context) ([]domainproduct.Product, error) {
	rows, err := r.queries.ListAllProducts(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]domainproduct.Product, len(rows))
	for i, row := range rows {
		result[i] = toDomainProduct(row)
	}
	return result, nil
}

func (r *SQLCRepository) Create(ctx context.Context, p domainproduct.Product) error {
	return r.queries.CreateProduct(ctx, sqldb.CreateProductParams{
		ID:                p.ID,
		Market:            string(p.Market),
		ExternalProductID: p.ExternalProductID,
		OriginalUrl:       p.OriginalURL,
		Title:             p.Title,
		MainImageUrl:      p.MainImageURL,
		CurrentPrice:      p.CurrentPrice,
		Currency:          p.Currency,
		ProductUrl:        p.ProductURL,
		CollectionSource:  p.CollectionSource,
		LastCollectedAt:   p.LastCollectedAt,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
	})
}

func (r *SQLCRepository) Update(ctx context.Context, p domainproduct.Product) error {
	_, err := r.queries.UpdateProduct(ctx, sqldb.UpdateProductParams{
		ID:               p.ID,
		OriginalUrl:      p.OriginalURL,
		Title:            p.Title,
		MainImageUrl:     p.MainImageURL,
		CurrentPrice:     p.CurrentPrice,
		Currency:         p.Currency,
		ProductUrl:       p.ProductURL,
		CollectionSource: p.CollectionSource,
		LastCollectedAt:  p.LastCollectedAt,
		UpdatedAt:        p.UpdatedAt,
	})
	return err
}

func toDomainProduct(row sqldb.GuguProduct) domainproduct.Product {
	return domainproduct.Product{
		ID:                row.ID,
		Market:            enum.Market(row.Market),
		ExternalProductID: row.ExternalProductID,
		OriginalURL:       row.OriginalUrl,
		Title:             row.Title,
		MainImageURL:      row.MainImageUrl,
		CurrentPrice:      row.CurrentPrice,
		Currency:          row.Currency,
		ProductURL:        row.ProductUrl,
		CollectionSource:  row.CollectionSource,
		LastCollectedAt:   row.LastCollectedAt,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}
