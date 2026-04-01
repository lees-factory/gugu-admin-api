-- name: CountAdminUsers :one
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
            SELECT MAX(oi.last_login_at)
            FROM gugu.oauth_identity oi
            WHERE oi.user_id = u.id
        ), '0001-01-01 00:00:00+00'::timestamptz) >= $4
    )
    OR (
        $3::text = 'INACTIVE'
        AND COALESCE((
            SELECT MAX(oi.last_login_at)
            FROM gugu.oauth_identity oi
            WHERE oi.user_id = u.id
        ), '0001-01-01 00:00:00+00'::timestamptz) < $4
    )
);

-- name: ListAdminUsers :many
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
        SELECT MAX(oi.last_login_at)
        FROM gugu.oauth_identity oi
        WHERE oi.user_id = u.id
    ) AS last_login_at,
    'FREE'::text AS plan,
    CASE
        WHEN COALESCE((
            SELECT MAX(oi.last_login_at)
            FROM gugu.oauth_identity oi
            WHERE oi.user_id = u.id
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
            SELECT MAX(oi.last_login_at)
            FROM gugu.oauth_identity oi
            WHERE oi.user_id = u.id
        ), '0001-01-01 00:00:00+00'::timestamptz) >= $4
    )
    OR (
        $3::text = 'INACTIVE'
        AND COALESCE((
            SELECT MAX(oi.last_login_at)
            FROM gugu.oauth_identity oi
            WHERE oi.user_id = u.id
        ), '0001-01-01 00:00:00+00'::timestamptz) < $4
    )
)
ORDER BY u.created_at DESC
LIMIT $5 OFFSET $6;
