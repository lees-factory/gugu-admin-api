package user

import (
	"context"
	"database/sql"
	"time"

	domainuser "github.com/ljj/gugu-admin-api/internal/core/domain/user"
	"github.com/ljj/gugu-admin-api/internal/storage/dbcore/sqldb"
)

type SQLCRepository struct {
	queries *sqldb.Queries
}

func NewSQLCRepository(db *sql.DB) *SQLCRepository {
	return &SQLCRepository{queries: sqldb.New(db)}
}

func (r *SQLCRepository) List(ctx context.Context, filter domainuser.ListFilter) ([]domainuser.User, error) {
	rows, err := r.queries.ListAdminUsers(ctx, sqldb.ListAdminUsersParams{
		Column1:     filter.Search,
		Column2:     string(filter.Plan),
		Column3:     string(filter.Status),
		LastLoginAt: activeAfter(),
		Limit:       filter.PageSize,
		Offset:      (filter.Page - 1) * filter.PageSize,
	})
	if err != nil {
		return nil, err
	}

	result := make([]domainuser.User, len(rows))
	for i, row := range rows {
		result[i] = domainuser.User{
			ID:               row.ID,
			Email:            row.Email,
			DisplayName:      row.DisplayName,
			Plan:             domainuser.Plan(row.Plan),
			Status:           domainuser.Status(row.Status),
			EmailVerified:    row.EmailVerified,
			TrackedItemCount: row.TrackedItemCount,
			CreatedAt:        row.CreatedAt,
			LastLoginAt:      toTimePtr(row.LastLoginAt),
		}
	}

	return result, nil
}

func (r *SQLCRepository) Count(ctx context.Context, filter domainuser.ListFilter) (int64, error) {
	return r.queries.CountAdminUsers(ctx, sqldb.CountAdminUsersParams{
		Column1:     filter.Search,
		Column2:     string(filter.Plan),
		Column3:     string(filter.Status),
		LastLoginAt: activeAfter(),
	})
}

func activeAfter() time.Time {
	return time.Now().AddDate(0, 0, -30)
}

func toTimePtr(value any) *time.Time {
	switch v := value.(type) {
	case time.Time:
		return &v
	case sql.NullTime:
		if !v.Valid {
			return nil
		}
		return &v.Time
	default:
		return nil
	}
}
