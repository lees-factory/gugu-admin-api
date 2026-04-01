package pricehistory

import (
	"context"
	"database/sql"
	"time"

	"github.com/ljj/gugu-admin-api/internal/storage/dbcore/sqldb"
)

type Repository struct {
	queries *sqldb.Queries
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{queries: sqldb.New(db)}
}

// --- Product Price History ---

func (r *Repository) InsertProductPrice(ctx context.Context, productID string, recordedAt time.Time, price, currency, changeValue string) error {
	return r.queries.InsertProductPriceHistory(ctx, sqldb.InsertProductPriceHistoryParams{
		ProductID:   productID,
		RecordedAt:  recordedAt,
		Price:       price,
		Currency:    currency,
		ChangeValue: changeValue,
	})
}

func (r *Repository) GetLatestProductPrice(ctx context.Context, productID, currency string) (string, error) {
	row, err := r.queries.GetLatestProductPrice(ctx, sqldb.GetLatestProductPriceParams{
		ProductID: productID,
		Currency:  currency,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return row.Price, nil
}

func (r *Repository) UpsertProductSnapshot(ctx context.Context, productID string, snapshotDate time.Time, price, currency string) error {
	return r.queries.UpsertProductPriceSnapshot(ctx, sqldb.UpsertProductPriceSnapshotParams{
		ProductID:    productID,
		SnapshotDate: snapshotDate,
		Price:        price,
		Currency:     currency,
	})
}

// --- SKU Price History ---

func (r *Repository) InsertSKUPrice(ctx context.Context, skuID string, recordedAt time.Time, price, currency, changeValue string) error {
	return r.queries.InsertSKUPriceHistory(ctx, sqldb.InsertSKUPriceHistoryParams{
		SkuID:       skuID,
		RecordedAt:  recordedAt,
		Price:       price,
		Currency:    currency,
		ChangeValue: changeValue,
	})
}

func (r *Repository) GetLatestSKUPrice(ctx context.Context, skuID, currency string) (string, error) {
	row, err := r.queries.GetLatestSKUPrice(ctx, sqldb.GetLatestSKUPriceParams{
		SkuID:    skuID,
		Currency: currency,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return row.Price, nil
}

func (r *Repository) UpsertSKUSnapshot(ctx context.Context, skuID string, snapshotDate time.Time, price, originalPrice, currency string) error {
	return r.queries.UpsertSKUPriceSnapshot(ctx, sqldb.UpsertSKUPriceSnapshotParams{
		SkuID:         skuID,
		SnapshotDate:  snapshotDate,
		Price:         price,
		OriginalPrice: originalPrice,
		Currency:      currency,
	})
}
