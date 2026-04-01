package product

import (
	"context"
	"database/sql"

	domainproduct "github.com/ljj/gugu-admin-api/internal/core/domain/product"
	"github.com/ljj/gugu-admin-api/internal/storage/dbcore/sqldb"
)

type SKUSQLCRepository struct {
	queries *sqldb.Queries
}

func NewSKUSQLCRepository(db *sql.DB) *SKUSQLCRepository {
	return &SKUSQLCRepository{queries: sqldb.New(db)}
}

func (r *SKUSQLCRepository) Create(ctx context.Context, sku domainproduct.SKU) error {
	return r.queries.CreateProductSKU(ctx, toCreateSKUParams(sku))
}

func (r *SKUSQLCRepository) Upsert(ctx context.Context, sku domainproduct.SKU) error {
	return r.queries.UpsertProductSKU(ctx, toUpsertSKUParams(sku))
}

func (r *SKUSQLCRepository) FindByID(ctx context.Context, skuID string) (*domainproduct.SKU, error) {
	row, err := r.queries.FindProductSKUByID(ctx, skuID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	s := toDomainSKU(row)
	return &s, nil
}

func (r *SKUSQLCRepository) FindByProductID(ctx context.Context, productID string) ([]domainproduct.SKU, error) {
	rows, err := r.queries.FindProductSKUsByProductID(ctx, productID)
	if err != nil {
		return nil, err
	}
	result := make([]domainproduct.SKU, len(rows))
	for i, row := range rows {
		result[i] = toDomainSKU(row)
	}
	return result, nil
}

func (r *SKUSQLCRepository) FindByProductIDAndExternalSKUID(ctx context.Context, productID string, externalSKUID string) (*domainproduct.SKU, error) {
	row, err := r.queries.FindProductSKUByProductIDAndExternalSKUID(ctx, sqldb.FindProductSKUByProductIDAndExternalSKUIDParams{
		ProductID:     productID,
		ExternalSkuID: externalSKUID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	s := toDomainSKU(row)
	return &s, nil
}

func (r *SKUSQLCRepository) CountByProductID(ctx context.Context, productID string) (int64, error) {
	return r.queries.CountSKUsByProductID(ctx, productID)
}

func toCreateSKUParams(s domainproduct.SKU) sqldb.CreateProductSKUParams {
	return sqldb.CreateProductSKUParams{
		ID:            s.ID,
		ProductID:     s.ProductID,
		ExternalSkuID: s.ExternalSKUID,
		OriginSkuID:   s.OriginSKUID,
		SkuName:       s.SKUName,
		Color:         s.Color,
		Size:          s.Size,
		Price:         s.Price,
		OriginalPrice: s.OriginalPrice,
		Currency:      s.Currency,
		ImageUrl:      s.ImageURL,
		SkuProperties: s.SKUProperties,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}

func toUpsertSKUParams(s domainproduct.SKU) sqldb.UpsertProductSKUParams {
	return sqldb.UpsertProductSKUParams{
		ID:            s.ID,
		ProductID:     s.ProductID,
		ExternalSkuID: s.ExternalSKUID,
		OriginSkuID:   s.OriginSKUID,
		SkuName:       s.SKUName,
		Color:         s.Color,
		Size:          s.Size,
		Price:         s.Price,
		OriginalPrice: s.OriginalPrice,
		Currency:      s.Currency,
		ImageUrl:      s.ImageURL,
		SkuProperties: s.SKUProperties,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}

func toDomainSKU(row sqldb.GuguSku) domainproduct.SKU {
	return domainproduct.SKU{
		ID:            row.ID,
		ProductID:     row.ProductID,
		ExternalSKUID: row.ExternalSkuID,
		OriginSKUID:   row.OriginSkuID,
		SKUName:       row.SkuName,
		Color:         row.Color,
		Size:          row.Size,
		Price:         row.Price,
		OriginalPrice: row.OriginalPrice,
		Currency:      row.Currency,
		ImageURL:      row.ImageUrl,
		SKUProperties: row.SkuProperties,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}

