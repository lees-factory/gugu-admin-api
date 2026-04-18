package adminauth

import (
	"context"
	"database/sql"
	"time"

	domainadminauth "github.com/ljj/gugu-admin-api/internal/core/domain/adminauth"
)

type SQLRepository struct {
	db *sql.DB
}

func NewSQLRepository(db *sql.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

func (r *SQLRepository) GetByID(ctx context.Context, id string) (*domainadminauth.AdminUser, error) {
	row := r.db.QueryRowContext(ctx, getAdminUserByIDQuery, id)
	return scanAdminUser(row)
}

func (r *SQLRepository) GetByLoginID(ctx context.Context, loginID string) (*domainadminauth.AdminUser, error) {
	row := r.db.QueryRowContext(ctx, getAdminUserByLoginIDQuery, loginID)
	return scanAdminUser(row)
}

func scanAdminUser(row *sql.Row) (*domainadminauth.AdminUser, error) {
	var user domainadminauth.AdminUser
	var lastLoginAt sql.NullTime
	if err := row.Scan(
		&user.ID,
		&user.LoginID,
		&user.PasswordHash,
		&user.Active,
		&lastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}
	return &user, nil
}

func (r *SQLRepository) UpdateLastLoginAt(ctx context.Context, id string, at time.Time) error {
	_, err := r.db.ExecContext(ctx, updateAdminUserLastLoginAtQuery, id, at)
	return err
}

const getAdminUserByIDQuery = `
SELECT
    id,
    login_id,
    password_hash,
    active,
    last_login_at,
    created_at,
    updated_at
FROM gugu.admin_user
WHERE id = $1
LIMIT 1;
`

const getAdminUserByLoginIDQuery = `
SELECT
    id,
    login_id,
    password_hash,
    active,
    last_login_at,
    created_at,
    updated_at
FROM gugu.admin_user
WHERE login_id = $1
LIMIT 1;
`

const updateAdminUserLastLoginAtQuery = `
UPDATE gugu.admin_user
SET
    last_login_at = $2,
    updated_at = $2
WHERE id = $1;
`
