package hotproduct

import (
	"context"
	"database/sql"
	"time"

	"github.com/ljj/gugu-admin-api/internal/storage/dbcore/sqldb"
)

type HotProductRow struct {
	ID                string
	ExternalProductID string
	Title             string
	ImageURL          string
	ProductURL        string
	SalePrice         string
	Currency          string
	CollectedDate     time.Time
	CreatedAt         time.Time
}

type SQLCRepository struct {
	queries *sqldb.Queries
}

func NewSQLCRepository(db *sql.DB) *SQLCRepository {
	return &SQLCRepository{queries: sqldb.New(db)}
}

func (r *SQLCRepository) Insert(ctx context.Context, row HotProductRow) error {
	return r.queries.InsertHotProduct(ctx, sqldb.InsertHotProductParams{
		ID:                row.ID,
		ExternalProductID: row.ExternalProductID,
		Title:             row.Title,
		ImageUrl:          row.ImageURL,
		ProductUrl:        row.ProductURL,
		SalePrice:         row.SalePrice,
		Currency:          row.Currency,
		CollectedDate:     row.CollectedDate,
		CreatedAt:         row.CreatedAt,
	})
}

func (r *SQLCRepository) ListByDate(ctx context.Context, date time.Time) ([]HotProductRow, error) {
	rows, err := r.queries.ListHotProductsByDate(ctx, date)
	if err != nil {
		return nil, err
	}
	return toRows(rows), nil
}

func (r *SQLCRepository) ListLatest(ctx context.Context) ([]HotProductRow, error) {
	rows, err := r.queries.ListHotProductsLatest(ctx)
	if err != nil {
		return nil, err
	}
	return toRows(rows), nil
}

func (r *SQLCRepository) DeleteBefore(ctx context.Context, date time.Time) (int64, error) {
	return r.queries.DeleteHotProductsBefore(ctx, date)
}

func toRows(rows []sqldb.GuguHotProduct) []HotProductRow {
	result := make([]HotProductRow, len(rows))
	for i, row := range rows {
		result[i] = HotProductRow{
			ID:                row.ID,
			ExternalProductID: row.ExternalProductID,
			Title:             row.Title,
			ImageURL:          row.ImageUrl,
			ProductURL:        row.ProductUrl,
			SalePrice:         row.SalePrice,
			Currency:          row.Currency,
			CollectedDate:     row.CollectedDate,
			CreatedAt:         row.CreatedAt,
		}
	}
	return result
}
