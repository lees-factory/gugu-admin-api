CREATE SCHEMA IF NOT EXISTS gugu;

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
    external_product_id TEXT NOT NULL,
    original_url TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    main_image_url TEXT NOT NULL DEFAULT '',
    current_price TEXT NOT NULL DEFAULT '',
    currency TEXT NOT NULL DEFAULT '',
    product_url TEXT NOT NULL DEFAULT '',
    collection_source TEXT NOT NULL DEFAULT '',
    last_collected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (market, external_product_id)
);

CREATE TABLE IF NOT EXISTS gugu.sku (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL REFERENCES gugu.product(id),
    external_sku_id TEXT NOT NULL DEFAULT '',
    origin_sku_id TEXT NOT NULL DEFAULT '',
    sku_name TEXT NOT NULL DEFAULT '',
    color TEXT NOT NULL DEFAULT '',
    size TEXT NOT NULL DEFAULT '',
    price TEXT NOT NULL DEFAULT '',
    original_price TEXT NOT NULL DEFAULT '',
    currency TEXT NOT NULL DEFAULT '',
    image_url TEXT NOT NULL DEFAULT '',
    sku_properties TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (product_id, external_sku_id)
);

CREATE TABLE IF NOT EXISTS gugu.product_price_history (
    product_id TEXT NOT NULL REFERENCES gugu.product(id),
    recorded_at TIMESTAMPTZ NOT NULL,
    price TEXT NOT NULL DEFAULT '',
    currency TEXT NOT NULL DEFAULT '',
    change_value TEXT NOT NULL DEFAULT '',
    sku_id TEXT,
    PRIMARY KEY (product_id, currency, recorded_at)
);

CREATE TABLE IF NOT EXISTS gugu.product_price_snapshot (
    product_id TEXT NOT NULL REFERENCES gugu.product(id),
    snapshot_date DATE NOT NULL,
    price TEXT NOT NULL DEFAULT '',
    currency TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (product_id, currency, snapshot_date)
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

CREATE TABLE IF NOT EXISTS gugu.user_tracked_item (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    product_id TEXT NOT NULL REFERENCES gugu.product(id),
    original_url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    sku_id TEXT
);

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
    app_type TEXT NOT NULL DEFAULT 'AFFILIATE'
);

CREATE TABLE IF NOT EXISTS gugu.hot_product (
    id TEXT PRIMARY KEY,
    external_product_id TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    image_url TEXT NOT NULL DEFAULT '',
    product_url TEXT NOT NULL DEFAULT '',
    promotion_link TEXT NOT NULL DEFAULT '',
    sale_price TEXT NOT NULL DEFAULT '',
    currency TEXT NOT NULL DEFAULT '',
    collected_date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (external_product_id, collected_date)
);

CREATE INDEX IF NOT EXISTS idx_hot_product_collected_date ON gugu.hot_product(collected_date);

CREATE INDEX IF NOT EXISTS idx_aliexpress_seller_token_app_type ON gugu.aliexpress_seller_token(app_type);

CREATE INDEX IF NOT EXISTS idx_product_market_external_product_id ON gugu.product(market, external_product_id);
CREATE INDEX IF NOT EXISTS idx_product_sku_product_id ON gugu.sku(product_id);
CREATE INDEX IF NOT EXISTS idx_app_user_email ON gugu.app_user(email);
CREATE INDEX IF NOT EXISTS idx_oauth_identity_user_id ON gugu.oauth_identity(user_id);
CREATE INDEX IF NOT EXISTS idx_user_tracked_item_user_id ON gugu.user_tracked_item(user_id);
