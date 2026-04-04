-- name: CreateProduct :exec
INSERT INTO gugu.product (
    id, market, external_product_id, original_url, title, main_image_url,
    current_price, currency, product_url, collection_source,
    last_collected_at, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
);

-- name: UpdateProduct :execrows
UPDATE gugu.product
SET original_url = $2, title = $3, main_image_url = $4,
    current_price = $5, currency = $6, product_url = $7,
    collection_source = $8, last_collected_at = $9, updated_at = $10
WHERE id = $1;

-- name: FindProductByID :one
SELECT id, market, external_product_id, original_url, title, main_image_url,
       current_price, currency, product_url, collection_source,
       last_collected_at, created_at, updated_at
FROM gugu.product
WHERE id = $1;

-- name: FindProductByMarketAndExternalProductID :one
SELECT id, market, external_product_id, original_url, title, main_image_url,
       current_price, currency, product_url, collection_source,
       last_collected_at, created_at, updated_at
FROM gugu.product
WHERE market = $1 AND external_product_id = $2;

-- name: FindProductsByIDs :many
SELECT id, market, external_product_id, original_url, title, main_image_url,
       current_price, currency, product_url, collection_source,
       last_collected_at, created_at, updated_at
FROM gugu.product
WHERE id = ANY($1::text[]);

-- name: ListProductsByMarket :many
SELECT id, market, external_product_id, original_url, title, main_image_url,
       current_price, currency, product_url, collection_source,
       last_collected_at, created_at, updated_at
FROM gugu.product
WHERE market = $1
ORDER BY created_at;

-- name: ListAllProducts :many
SELECT id, market, external_product_id, original_url, title, main_image_url,
       current_price, currency, product_url, collection_source,
       last_collected_at, created_at, updated_at
FROM gugu.product
ORDER BY created_at;

-- name: ListAllLocalizedProducts :many
SELECT
    p.id,
    p.market,
    p.external_product_id,
    p.original_url,
    COALESCE(pv.title, p.title) AS title,
    COALESCE(pv.main_image_url, p.main_image_url) AS main_image_url,
    COALESCE(pv.current_price, p.current_price) AS current_price,
    COALESCE(pv.currency, p.currency) AS currency,
    COALESCE(pv.product_url, p.product_url) AS product_url,
    p.collection_source,
    COALESCE(pv.last_collected_at, p.last_collected_at) AS last_collected_at,
    p.created_at,
    GREATEST(p.updated_at, COALESCE(pv.updated_at, p.updated_at))::timestamptz AS updated_at
FROM gugu.product p
LEFT JOIN gugu.product_variant pv
    ON pv.product_id = p.id
   AND pv.language = $1
   AND pv.currency = $2
ORDER BY p.created_at;

-- name: ListProductsWithoutSKUs :many
SELECT p.id, p.market, p.external_product_id, p.original_url, p.title,
       p.main_image_url, p.current_price, p.currency, p.product_url,
       p.collection_source, p.last_collected_at, p.created_at, p.updated_at
FROM gugu.product p
LEFT JOIN gugu.sku s ON p.id = s.product_id
WHERE s.id IS NULL
ORDER BY p.created_at;

-- name: ListProductsByCollectionSource :many
SELECT id, market, external_product_id, original_url, title, main_image_url,
       current_price, currency, product_url, collection_source,
       last_collected_at, created_at, updated_at
FROM gugu.product
WHERE collection_source = $1
ORDER BY created_at;

-- name: ListLocalizedProductsByCollectionSource :many
SELECT
    p.id,
    p.market,
    p.external_product_id,
    p.original_url,
    COALESCE(pv.title, p.title) AS title,
    COALESCE(pv.main_image_url, p.main_image_url) AS main_image_url,
    COALESCE(pv.current_price, p.current_price) AS current_price,
    COALESCE(pv.currency, p.currency) AS currency,
    COALESCE(pv.product_url, p.product_url) AS product_url,
    p.collection_source,
    COALESCE(pv.last_collected_at, p.last_collected_at) AS last_collected_at,
    p.created_at,
    GREATEST(p.updated_at, COALESCE(pv.updated_at, p.updated_at))::timestamptz AS updated_at
FROM gugu.product p
LEFT JOIN gugu.product_variant pv
    ON pv.product_id = p.id
   AND pv.language = $2
   AND pv.currency = $3
WHERE p.collection_source = $1
ORDER BY p.created_at;

-- name: ListPriceUpdateCandidateProducts :many
SELECT id, market, external_product_id, original_url, title, main_image_url,
       current_price, currency, product_url, collection_source,
       last_collected_at, created_at, updated_at
FROM gugu.product
WHERE ($1::text = '' OR collection_source = $1)
  AND ($2::text = '' OR market = $2)
  AND ($3::timestamptz = '0001-01-01 00:00:00+00'::timestamptz OR last_collected_at <= $3)
ORDER BY last_collected_at, created_at;
