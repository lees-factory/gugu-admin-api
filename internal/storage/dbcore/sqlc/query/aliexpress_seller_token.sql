-- name: GetTokenByAppType :one
SELECT id, seller_id, havana_id, app_user_id, user_nick, account,
       account_platform, locale, sp, access_token, refresh_token,
       access_token_expires_at, refresh_token_expires_at,
       last_refreshed_at, authorized_at, created_at, updated_at, app_type
FROM gugu.aliexpress_seller_token
WHERE app_type = $1
ORDER BY updated_at DESC
LIMIT 1;

-- name: GetTokenBySellerID :one
SELECT id, seller_id, havana_id, app_user_id, user_nick, account,
       account_platform, locale, sp, access_token, refresh_token,
       access_token_expires_at, refresh_token_expires_at,
       last_refreshed_at, authorized_at, created_at, updated_at, app_type
FROM gugu.aliexpress_seller_token
WHERE seller_id = $1;

-- name: UpsertToken :exec
INSERT INTO gugu.aliexpress_seller_token (
    id, seller_id, havana_id, app_user_id, user_nick, account,
    account_platform, locale, sp, access_token, refresh_token,
    access_token_expires_at, refresh_token_expires_at,
    last_refreshed_at, authorized_at, created_at, updated_at, app_type
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
) ON CONFLICT (seller_id) DO UPDATE SET
    havana_id = EXCLUDED.havana_id,
    app_user_id = EXCLUDED.app_user_id,
    user_nick = EXCLUDED.user_nick,
    account = EXCLUDED.account,
    account_platform = EXCLUDED.account_platform,
    locale = EXCLUDED.locale,
    sp = EXCLUDED.sp,
    access_token = EXCLUDED.access_token,
    refresh_token = EXCLUDED.refresh_token,
    access_token_expires_at = EXCLUDED.access_token_expires_at,
    refresh_token_expires_at = EXCLUDED.refresh_token_expires_at,
    last_refreshed_at = EXCLUDED.last_refreshed_at,
    updated_at = EXCLUDED.updated_at,
    app_type = EXCLUDED.app_type;

-- name: GetExpiringSoonTokens :many
SELECT id, seller_id, havana_id, app_user_id, user_nick, account,
       account_platform, locale, sp, access_token, refresh_token,
       access_token_expires_at, refresh_token_expires_at,
       last_refreshed_at, authorized_at, created_at, updated_at, app_type
FROM gugu.aliexpress_seller_token
WHERE access_token_expires_at <= $1
ORDER BY access_token_expires_at;
