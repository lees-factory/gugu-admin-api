CREATE SCHEMA IF NOT EXISTS gugu;

CREATE TABLE IF NOT EXISTS gugu.aliexpress_seller_token (
    id TEXT PRIMARY KEY,
    seller_id TEXT NOT NULL UNIQUE,
    havana_id TEXT NOT NULL DEFAULT '',
    app_user_id TEXT NOT NULL DEFAULT '',
    user_nick TEXT NOT NULL DEFAULT '',
    account TEXT NOT NULL DEFAULT '',
    account_platform TEXT NOT NULL DEFAULT '',
    locale TEXT NOT NULL DEFAULT '',
    sp TEXT NOT NULL DEFAULT '',
    access_token TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    access_token_expires_at TIMESTAMPTZ NOT NULL,
    refresh_token_expires_at TIMESTAMPTZ,
    last_refreshed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    authorized_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    app_type TEXT NOT NULL DEFAULT 'AFFILIATE',
    UNIQUE (app_type)
);

CREATE TABLE IF NOT EXISTS gugu.app_user (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL DEFAULT '',
    auth_source TEXT NOT NULL,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    email_verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS gugu.email_verification (
    code TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    email TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS gugu.oauth_identity (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    subject TEXT NOT NULL,
    email TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS gugu.product (
    id TEXT PRIMARY KEY,
    market TEXT NOT NULL,
    original_url TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    main_image_url TEXT NOT NULL DEFAULT '',
    product_url TEXT NOT NULL DEFAULT '',
    collection_source TEXT NOT NULL DEFAULT '',
    last_collected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    promotion_link TEXT NOT NULL DEFAULT '',
    origin_product_id TEXT NOT NULL,
    UNIQUE (market, origin_product_id)
);

CREATE TABLE IF NOT EXISTS gugu.product_external_alias (
    id TEXT PRIMARY KEY,
    market TEXT NOT NULL,
    alias_external_product_id TEXT NOT NULL,
    product_id TEXT NOT NULL REFERENCES gugu.product(id),
    alias_type TEXT NOT NULL DEFAULT 'VIEW',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (market, alias_external_product_id)
);

CREATE TABLE IF NOT EXISTS gugu.product_localization (
    product_id TEXT NOT NULL REFERENCES gugu.product(id),
    language TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    main_image_url TEXT NOT NULL DEFAULT '',
    product_url TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (product_id, language)
);

CREATE TABLE IF NOT EXISTS gugu.sku (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL REFERENCES gugu.product(id),
    external_sku_id TEXT NOT NULL DEFAULT '',
    sku_name TEXT NOT NULL DEFAULT '',
    color TEXT NOT NULL DEFAULT '',
    size TEXT NOT NULL DEFAULT '',
    image_url TEXT NOT NULL DEFAULT '',
    sku_properties TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    origin_sku_id TEXT NOT NULL DEFAULT '',
    UNIQUE (product_id, external_sku_id)
);

CREATE TABLE IF NOT EXISTS gugu.sku_localization (
    sku_id TEXT NOT NULL REFERENCES gugu.sku(id),
    language TEXT NOT NULL,
    sku_name TEXT NOT NULL DEFAULT '',
    color_name TEXT NOT NULL DEFAULT '',
    size_name TEXT NOT NULL DEFAULT '',
    sku_properties TEXT NOT NULL DEFAULT '',
    image_url TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (sku_id, language)
);

CREATE TABLE IF NOT EXISTS gugu.sku_price_history (
    sku_id TEXT NOT NULL REFERENCES gugu.sku(id),
    recorded_at TIMESTAMPTZ NOT NULL,
    price TEXT NOT NULL DEFAULT '',
    currency TEXT NOT NULL DEFAULT '',
    change_value TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (sku_id, currency, recorded_at)
);

CREATE TABLE IF NOT EXISTS gugu.sku_price_snapshot (
    sku_id TEXT NOT NULL REFERENCES gugu.sku(id),
    snapshot_date DATE NOT NULL,
    price TEXT NOT NULL DEFAULT '',
    original_price TEXT NOT NULL DEFAULT '',
    currency TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (sku_id, currency, snapshot_date)
);

CREATE TABLE IF NOT EXISTS gugu.sku_snapshot_ingest_run (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL REFERENCES gugu.product(id),
    currency TEXT NOT NULL,
    snapshot_date DATE NOT NULL,
    expected_sku_count INTEGER NOT NULL DEFAULT 0,
    collected_sku_count INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    error_message TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS gugu.sku_price_snapshot_staging (
    run_id TEXT NOT NULL REFERENCES gugu.sku_snapshot_ingest_run(id),
    sku_id TEXT NOT NULL REFERENCES gugu.sku(id),
    snapshot_date DATE NOT NULL,
    price TEXT NOT NULL DEFAULT '',
    original_price TEXT NOT NULL DEFAULT '',
    currency TEXT NOT NULL,
    PRIMARY KEY (run_id, sku_id, currency)
);

CREATE TABLE IF NOT EXISTS gugu.price_alert (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES gugu.app_user(id),
    sku_id TEXT NOT NULL REFERENCES gugu.sku(id),
    channel TEXT NOT NULL DEFAULT 'EMAIL',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS gugu.user_login_session (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES gugu.app_user(id),
    refresh_token_hash TEXT NOT NULL,
    token_family_id TEXT NOT NULL,
    parent_session_id TEXT REFERENCES gugu.user_login_session(id),
    user_agent TEXT NOT NULL DEFAULT '',
    client_ip TEXT NOT NULL DEFAULT '',
    device_name TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    rotated_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    reuse_detected_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS gugu.user_tracked_item (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES gugu.app_user(id),
    product_id TEXT NOT NULL REFERENCES gugu.product(id),
    original_url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    sku_id TEXT,
    currency TEXT NOT NULL DEFAULT 'KRW',
    view_external_product_id TEXT NOT NULL DEFAULT '',
    preferred_language TEXT NOT NULL DEFAULT 'KO',
    tracking_scope TEXT NOT NULL DEFAULT 'PRODUCT_ALL_SKU'
);

CREATE TABLE IF NOT EXISTS gugu.user_tracked_item_watch_sku (
    tracked_item_id TEXT NOT NULL REFERENCES gugu.user_tracked_item(id),
    sku_id TEXT NOT NULL REFERENCES gugu.sku(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tracked_item_id, sku_id)
);

CREATE INDEX IF NOT EXISTS idx_aliexpress_seller_token_app_type ON gugu.aliexpress_seller_token(app_type);
CREATE INDEX IF NOT EXISTS idx_product_market_origin_product_id ON gugu.product(market, origin_product_id);
CREATE INDEX IF NOT EXISTS idx_product_external_alias_product_id ON gugu.product_external_alias(product_id);
CREATE INDEX IF NOT EXISTS idx_product_sku_product_id ON gugu.sku(product_id);
CREATE INDEX IF NOT EXISTS idx_app_user_email ON gugu.app_user(email);
CREATE INDEX IF NOT EXISTS idx_oauth_identity_user_id ON gugu.oauth_identity(user_id);
CREATE INDEX IF NOT EXISTS idx_user_tracked_item_user_id ON gugu.user_tracked_item(user_id);
