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
		result[i] = toDomainProductByIDs(row)
	}
	return result, nil
}

func (r *SQLCRepository) FindByMarketAndExternalProductID(ctx context.Context, market enum.Market, externalProductID string) (*domainproduct.Product, error) {
	row, err := r.queries.FindProductByMarketAndExternalProductID(ctx, sqldb.FindProductByMarketAndExternalProductIDParams{
		Market:          string(market),
		OriginProductID: externalProductID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	p := toDomainProductByExternalID(row)
	return &p, nil
}

func (r *SQLCRepository) ListActiveTrackedProductIDs(ctx context.Context) ([]string, error) {
	return r.queries.ListActiveTrackedProductIDs(ctx)
}

func (r *SQLCRepository) ListByMarket(ctx context.Context, market enum.Market) ([]domainproduct.Product, error) {
	rows, err := r.queries.ListProductsByMarket(ctx, string(market))
	if err != nil {
		return nil, err
	}
	result := make([]domainproduct.Product, len(rows))
	for i, row := range rows {
		result[i] = toDomainProductByMarket(row)
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
		result[i] = toDomainProductByCollectionSource(row)
	}
	return result, nil
}

func (r *SQLCRepository) ListAllLocalized(ctx context.Context, language string) ([]domainproduct.LocalizedProduct, error) {
	rows, err := r.queries.ListAllLocalizedProducts(ctx, language)
	if err != nil {
		return nil, err
	}
	result := make([]domainproduct.LocalizedProduct, len(rows))
	for i, row := range rows {
		result[i] = toDomainLocalizedProductFromListAll(row, language)
	}
	return result, nil
}

func (r *SQLCRepository) ListByCollectionSourceLocalized(ctx context.Context, collectionSource, language string) ([]domainproduct.LocalizedProduct, error) {
	rows, err := r.queries.ListLocalizedProductsByCollectionSource(ctx, sqldb.ListLocalizedProductsByCollectionSourceParams{
		CollectionSource: collectionSource,
		Language:         language,
	})
	if err != nil {
		return nil, err
	}
	result := make([]domainproduct.LocalizedProduct, len(rows))
	for i, row := range rows {
		result[i] = toDomainLocalizedProductFromCollection(row, language)
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
		result[i] = toDomainPriceUpdateCandidate(row)
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
		result[i] = toDomainProductListAll(row)
	}
	return result, nil
}

func (r *SQLCRepository) Create(ctx context.Context, p domainproduct.Product) error {
	return r.queries.CreateProduct(ctx, sqldb.CreateProductParams{
		ID:               p.ID,
		Market:           string(p.Market),
		OriginProductID:  p.ExternalProductID,
		OriginalUrl:      p.OriginalURL,
		Title:            p.Title,
		MainImageUrl:     p.MainImageURL,
		ProductUrl:       p.ProductURL,
		CollectionSource: p.CollectionSource,
		LastCollectedAt:  p.LastCollectedAt,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	})
}

func (r *SQLCRepository) Update(ctx context.Context, p domainproduct.Product) error {
	_, err := r.queries.UpdateProduct(ctx, sqldb.UpdateProductParams{
		ID:               p.ID,
		OriginalUrl:      p.OriginalURL,
		Title:            p.Title,
		MainImageUrl:     p.MainImageURL,
		ProductUrl:       p.ProductURL,
		CollectionSource: p.CollectionSource,
		LastCollectedAt:  p.LastCollectedAt,
		UpdatedAt:        p.UpdatedAt,
	})
	return err
}

func toDomainProduct(row sqldb.FindProductByIDRow) domainproduct.Product {
	return domainproduct.Product{
		ID:                row.ID,
		Market:            enum.Market(row.Market),
		ExternalProductID: row.ExternalProductID,
		OriginalURL:       row.OriginalUrl,
		Title:             row.Title,
		MainImageURL:      row.MainImageUrl,
		ProductURL:        row.ProductUrl,
		CollectionSource:  row.CollectionSource,
		LastCollectedAt:   row.LastCollectedAt,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func toDomainProductByExternalID(row sqldb.FindProductByMarketAndExternalProductIDRow) domainproduct.Product {
	return domainproduct.Product{
		ID:                row.ID,
		Market:            enum.Market(row.Market),
		ExternalProductID: row.ExternalProductID,
		OriginalURL:       row.OriginalUrl,
		Title:             row.Title,
		MainImageURL:      row.MainImageUrl,
		ProductURL:        row.ProductUrl,
		CollectionSource:  row.CollectionSource,
		LastCollectedAt:   row.LastCollectedAt,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func toDomainProductByIDs(row sqldb.FindProductsByIDsRow) domainproduct.Product {
	return domainproduct.Product{
		ID:                row.ID,
		Market:            enum.Market(row.Market),
		ExternalProductID: row.ExternalProductID,
		OriginalURL:       row.OriginalUrl,
		Title:             row.Title,
		MainImageURL:      row.MainImageUrl,
		ProductURL:        row.ProductUrl,
		CollectionSource:  row.CollectionSource,
		LastCollectedAt:   row.LastCollectedAt,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func toDomainProductListAll(row sqldb.ListAllProductsRow) domainproduct.Product {
	return domainproduct.Product{
		ID:                row.ID,
		Market:            enum.Market(row.Market),
		ExternalProductID: row.ExternalProductID,
		OriginalURL:       row.OriginalUrl,
		Title:             row.Title,
		MainImageURL:      row.MainImageUrl,
		ProductURL:        row.ProductUrl,
		CollectionSource:  row.CollectionSource,
		LastCollectedAt:   row.LastCollectedAt,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func toDomainProductByMarket(row sqldb.ListProductsByMarketRow) domainproduct.Product {
	return domainproduct.Product{
		ID:                row.ID,
		Market:            enum.Market(row.Market),
		ExternalProductID: row.ExternalProductID,
		OriginalURL:       row.OriginalUrl,
		Title:             row.Title,
		MainImageURL:      row.MainImageUrl,
		ProductURL:        row.ProductUrl,
		CollectionSource:  row.CollectionSource,
		LastCollectedAt:   row.LastCollectedAt,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func toDomainProductByCollectionSource(row sqldb.ListProductsByCollectionSourceRow) domainproduct.Product {
	return domainproduct.Product{
		ID:                row.ID,
		Market:            enum.Market(row.Market),
		ExternalProductID: row.ExternalProductID,
		OriginalURL:       row.OriginalUrl,
		Title:             row.Title,
		MainImageURL:      row.MainImageUrl,
		ProductURL:        row.ProductUrl,
		CollectionSource:  row.CollectionSource,
		LastCollectedAt:   row.LastCollectedAt,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func toDomainPriceUpdateCandidate(row sqldb.ListPriceUpdateCandidateProductsRow) domainproduct.Product {
	return domainproduct.Product{
		ID:                row.ID,
		Market:            enum.Market(row.Market),
		ExternalProductID: row.ExternalProductID,
		OriginalURL:       row.OriginalUrl,
		Title:             row.Title,
		MainImageURL:      row.MainImageUrl,
		ProductURL:        row.ProductUrl,
		CollectionSource:  row.CollectionSource,
		LastCollectedAt:   row.LastCollectedAt,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func toDomainLocalizedProductFromListAll(row sqldb.ListAllLocalizedProductsRow, language string) domainproduct.LocalizedProduct {
	return domainproduct.LocalizedProduct{
		ID:                row.ID,
		Market:            enum.Market(row.Market),
		ExternalProductID: row.ExternalProductID,
		OriginalURL:       row.OriginalUrl,
		Title:             row.Title,
		MainImageURL:      row.MainImageUrl,
		ProductURL:        row.ProductUrl,
		CollectionSource:  row.CollectionSource,
		LastCollectedAt:   row.LastCollectedAt,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
		Language:          language,
	}
}

func toDomainLocalizedProductFromCollection(row sqldb.ListLocalizedProductsByCollectionSourceRow, language string) domainproduct.LocalizedProduct {
	return domainproduct.LocalizedProduct{
		ID:                row.ID,
		Market:            enum.Market(row.Market),
		ExternalProductID: row.ExternalProductID,
		OriginalURL:       row.OriginalUrl,
		Title:             row.Title,
		MainImageURL:      row.MainImageUrl,
		ProductURL:        row.ProductUrl,
		CollectionSource:  row.CollectionSource,
		LastCollectedAt:   row.LastCollectedAt,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
		Language:          language,
	}
}
