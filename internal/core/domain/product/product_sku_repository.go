package product

import "context"

type SKURepository interface {
	Create(ctx context.Context, sku SKU) error
	Upsert(ctx context.Context, sku SKU) error
	FindByID(ctx context.Context, skuID string) (*SKU, error)
	FindByProductID(ctx context.Context, productID string) ([]SKU, error)
	FindByProductIDAndExternalSKUID(ctx context.Context, productID string, externalSKUID string) (*SKU, error)
	CountByProductID(ctx context.Context, productID string) (int64, error)
}
