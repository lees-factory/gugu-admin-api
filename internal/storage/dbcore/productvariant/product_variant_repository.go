package productvariant

import (
	"context"
	"database/sql"
	"time"

	"github.com/ljj/gugu-admin-api/internal/storage/dbcore/sqldb"
)

type VariantRow struct {
	ProductID       string
	Language        string
	Currency        string
	Title           string
	MainImageURL    string
	ProductURL      string
	CurrentPrice    string
	LastCollectedAt time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type SQLCRepository struct {
	queries *sqldb.Queries
}

func NewSQLCRepository(db *sql.DB) *SQLCRepository {
	return &SQLCRepository{queries: sqldb.New(db)}
}

func (r *SQLCRepository) Upsert(ctx context.Context, row VariantRow) error {
	return r.queries.UpsertProductVariant(ctx, sqldb.UpsertProductVariantParams{
		ProductID:       row.ProductID,
		Language:        row.Language,
		Currency:        row.Currency,
		Title:           row.Title,
		MainImageUrl:    row.MainImageURL,
		ProductUrl:      row.ProductURL,
		CurrentPrice:    row.CurrentPrice,
		LastCollectedAt: sql.NullTime{Time: row.LastCollectedAt, Valid: !row.LastCollectedAt.IsZero()},
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	})
}

func (r *SQLCRepository) UpsertProductVariant(ctx context.Context, productID, language, currency, title, mainImageURL, productURL, currentPrice string, collectedAt time.Time) error {
	now := time.Now()
	return r.Upsert(ctx, VariantRow{
		ProductID:       productID,
		Language:        language,
		Currency:        currency,
		Title:           title,
		MainImageURL:    mainImageURL,
		ProductURL:      productURL,
		CurrentPrice:    currentPrice,
		LastCollectedAt: collectedAt,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
}
