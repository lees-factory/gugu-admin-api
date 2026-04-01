package token

import (
	"context"
	"database/sql"
	"time"

	domaintoken "github.com/ljj/gugu-admin-api/internal/core/domain/token"
	"github.com/ljj/gugu-admin-api/internal/storage/dbcore/sqldb"
)

type SQLCRepository struct {
	queries *sqldb.Queries
}

func NewSQLCRepository(db *sql.DB) *SQLCRepository {
	return &SQLCRepository{queries: sqldb.New(db)}
}

func (r *SQLCRepository) GetByAppType(ctx context.Context, appType domaintoken.AppType) (*domaintoken.SellerToken, error) {
	row, err := r.queries.GetTokenByAppType(ctx, string(appType))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t := toDomain(row)
	return &t, nil
}

func (r *SQLCRepository) GetBySellerID(ctx context.Context, sellerID string) (*domaintoken.SellerToken, error) {
	row, err := r.queries.GetTokenBySellerID(ctx, sellerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t := toDomain(row)
	return &t, nil
}

func (r *SQLCRepository) GetExpiringSoon(ctx context.Context, threshold time.Time) ([]domaintoken.SellerToken, error) {
	rows, err := r.queries.GetExpiringSoonTokens(ctx, threshold)
	if err != nil {
		return nil, err
	}
	result := make([]domaintoken.SellerToken, len(rows))
	for i, row := range rows {
		result[i] = toDomain(row)
	}
	return result, nil
}

func (r *SQLCRepository) Upsert(ctx context.Context, t domaintoken.SellerToken) error {
	var refreshExpiresAt sql.NullTime
	if t.RefreshTokenExpiresAt != nil {
		refreshExpiresAt = sql.NullTime{Time: *t.RefreshTokenExpiresAt, Valid: true}
	}

	return r.queries.UpsertToken(ctx, sqldb.UpsertTokenParams{
		ID:                    t.ID,
		SellerID:              t.SellerID,
		HavanaID:              t.HavanaID,
		AppUserID:             t.AppUserID,
		UserNick:              t.UserNick,
		Account:               t.Account,
		AccountPlatform:       t.AccountPlatform,
		Locale:                t.Locale,
		Sp:                    t.SP,
		AccessToken:           t.AccessToken,
		RefreshToken:          t.RefreshToken,
		AccessTokenExpiresAt:  t.AccessTokenExpiresAt,
		RefreshTokenExpiresAt: refreshExpiresAt,
		LastRefreshedAt:       t.LastRefreshedAt,
		AuthorizedAt:          t.AuthorizedAt,
		CreatedAt:             t.CreatedAt,
		UpdatedAt:             t.UpdatedAt,
		AppType:               string(t.AppType),
	})
}

func toDomain(row sqldb.GuguAliexpressSellerToken) domaintoken.SellerToken {
	t := domaintoken.SellerToken{
		ID:                   row.ID,
		SellerID:             row.SellerID,
		HavanaID:             row.HavanaID,
		AppUserID:            row.AppUserID,
		UserNick:             row.UserNick,
		Account:              row.Account,
		AccountPlatform:      row.AccountPlatform,
		Locale:               row.Locale,
		SP:                   row.Sp,
		AccessToken:          row.AccessToken,
		RefreshToken:         row.RefreshToken,
		AccessTokenExpiresAt: row.AccessTokenExpiresAt,
		LastRefreshedAt:      row.LastRefreshedAt,
		AuthorizedAt:         row.AuthorizedAt,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
		AppType:              domaintoken.AppType(row.AppType),
	}
	if row.RefreshTokenExpiresAt.Valid {
		t.RefreshTokenExpiresAt = &row.RefreshTokenExpiresAt.Time
	}
	return t
}
