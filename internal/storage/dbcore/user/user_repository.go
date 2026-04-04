package user

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
	domainuser "github.com/ljj/gugu-admin-api/internal/core/domain/user"
)

type SQLCRepository struct {
	db *sql.DB
}

func NewSQLCRepository(db *sql.DB) *SQLCRepository {
	return &SQLCRepository{db: db}
}

func (r *SQLCRepository) List(ctx context.Context, filter domainuser.ListFilter) ([]domainuser.User, error) {
	rows, err := r.db.QueryContext(ctx, listAdminUsersQuery,
		filter.Search,
		string(filter.Plan),
		string(filter.Status),
		activeAfter(),
		filter.PageSize,
		(filter.Page-1)*filter.PageSize,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domainuser.User
	var userIDs []string
	for rows.Next() {
		var user domainuser.User
		var plan string
		var status string
		var lastLoginAt sql.NullTime
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.DisplayName,
			&user.EmailVerified,
			&user.CreatedAt,
			&user.TrackedItemCount,
			&lastLoginAt,
			&plan,
			&status,
		); err != nil {
			return nil, err
		}
		user.Plan = domainuser.Plan(plan)
		user.Status = domainuser.Status(status)
		user.LastLoginAt = toNullTimePtr(lastLoginAt)
		result = append(result, user)
		userIDs = append(userIDs, user.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sessionsByUserID, err := r.listLoginSessionsByUserIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	for i := range result {
		result[i].Sessions = sessionsByUserID[result[i].ID]
	}

	return result, nil
}

func (r *SQLCRepository) Count(ctx context.Context, filter domainuser.ListFilter) (int64, error) {
	row := r.db.QueryRowContext(ctx, countAdminUsersQuery,
		filter.Search,
		string(filter.Plan),
		string(filter.Status),
		activeAfter(),
	)
	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func activeAfter() time.Time {
	return time.Now().AddDate(0, 0, -30)
}

func (r *SQLCRepository) listLoginSessionsByUserIDs(ctx context.Context, userIDs []string) (map[string][]domainuser.LoginSession, error) {
	result := make(map[string][]domainuser.LoginSession, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}

	rows, err := r.db.QueryContext(ctx, listLoginSessionsByUserIDsQuery, pq.Array(userIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var session domainuser.LoginSession
		var parentSessionID sql.NullString
		var rotatedAt sql.NullTime
		var revokedAt sql.NullTime
		var reuseDetectedAt sql.NullTime
		if err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.RefreshTokenHash,
			&session.TokenFamilyID,
			&parentSessionID,
			&session.UserAgent,
			&session.ClientIP,
			&session.DeviceName,
			&session.ExpiresAt,
			&session.LastSeenAt,
			&rotatedAt,
			&revokedAt,
			&reuseDetectedAt,
			&session.CreatedAt,
		); err != nil {
			return nil, err
		}
		session.ParentSessionID = toNullStringPtr(parentSessionID)
		session.RotatedAt = toNullTimePtr(rotatedAt)
		session.RevokedAt = toNullTimePtr(revokedAt)
		session.ReuseDetectedAt = toNullTimePtr(reuseDetectedAt)
		result[session.UserID] = append(result[session.UserID], session)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func toNullTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func toNullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

const countAdminUsersQuery = `
SELECT COUNT(*)
FROM gugu.app_user u
WHERE (
    $1::text = ''
    OR u.email ILIKE '%' || $1 || '%'
    OR u.display_name ILIKE '%' || $1 || '%'
)
AND (
    $2::text = ''
    OR ($2::text = 'FREE')
)
AND (
    $3::text = ''
    OR (
        $3::text = 'ACTIVE'
        AND COALESCE((
            SELECT MAX(uls.last_seen_at)
            FROM gugu.user_login_session uls
            WHERE uls.user_id = u.id
        ), '0001-01-01 00:00:00+00'::timestamptz) >= $4
    )
    OR (
        $3::text = 'INACTIVE'
        AND COALESCE((
            SELECT MAX(uls.last_seen_at)
            FROM gugu.user_login_session uls
            WHERE uls.user_id = u.id
        ), '0001-01-01 00:00:00+00'::timestamptz) < $4
    )
)`

const listAdminUsersQuery = `
SELECT
    u.id,
    u.email,
    u.display_name,
    u.email_verified,
    u.created_at,
    COALESCE((
        SELECT COUNT(*)::bigint
        FROM gugu.user_tracked_item uti
        WHERE uti.user_id = u.id
          AND uti.deleted_at IS NULL
    ), 0)::bigint AS tracked_item_count,
    (
        SELECT MAX(uls.last_seen_at)
        FROM gugu.user_login_session uls
        WHERE uls.user_id = u.id
    ) AS last_login_at,
    'FREE'::text AS plan,
    CASE
        WHEN COALESCE((
            SELECT MAX(uls.last_seen_at)
            FROM gugu.user_login_session uls
            WHERE uls.user_id = u.id
        ), '0001-01-01 00:00:00+00'::timestamptz) >= $4 THEN 'ACTIVE'::text
        ELSE 'INACTIVE'::text
    END AS status
FROM gugu.app_user u
WHERE (
    $1::text = ''
    OR u.email ILIKE '%' || $1 || '%'
    OR u.display_name ILIKE '%' || $1 || '%'
)
AND (
    $2::text = ''
    OR ($2::text = 'FREE')
)
AND (
    $3::text = ''
    OR (
        $3::text = 'ACTIVE'
        AND COALESCE((
            SELECT MAX(uls.last_seen_at)
            FROM gugu.user_login_session uls
            WHERE uls.user_id = u.id
        ), '0001-01-01 00:00:00+00'::timestamptz) >= $4
    )
    OR (
        $3::text = 'INACTIVE'
        AND COALESCE((
            SELECT MAX(uls.last_seen_at)
            FROM gugu.user_login_session uls
            WHERE uls.user_id = u.id
        ), '0001-01-01 00:00:00+00'::timestamptz) < $4
    )
)
ORDER BY u.created_at DESC
LIMIT $5 OFFSET $6`

const listLoginSessionsByUserIDsQuery = `
SELECT
    id,
    user_id,
    refresh_token_hash,
    token_family_id,
    parent_session_id,
    user_agent,
    client_ip,
    device_name,
    expires_at,
    last_seen_at,
    rotated_at,
    revoked_at,
    reuse_detected_at,
    created_at
FROM gugu.user_login_session
WHERE user_id = ANY($1::text[])
ORDER BY created_at DESC`
