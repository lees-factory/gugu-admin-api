-- name: CreateProductSKU :exec
INSERT INTO gugu.sku (
    id, product_id, external_sku_id, origin_sku_id, sku_name,
    color, size, price, original_price, currency,
    image_url, sku_properties, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
);

-- name: UpsertProductSKU :exec
INSERT INTO gugu.sku (
    id, product_id, external_sku_id, origin_sku_id, sku_name,
    color, size, price, original_price, currency,
    image_url, sku_properties, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
ON CONFLICT (product_id, external_sku_id) DO UPDATE SET
    origin_sku_id = EXCLUDED.origin_sku_id,
    sku_name = EXCLUDED.sku_name,
    color = EXCLUDED.color,
    size = EXCLUDED.size,
    price = EXCLUDED.price,
    original_price = EXCLUDED.original_price,
    currency = EXCLUDED.currency,
    image_url = EXCLUDED.image_url,
    sku_properties = EXCLUDED.sku_properties,
    updated_at = EXCLUDED.updated_at;

-- name: FindProductSKUByID :one
SELECT id, product_id, external_sku_id, origin_sku_id, sku_name,
       color, size, price, original_price, currency,
       image_url, sku_properties, created_at, updated_at
FROM gugu.sku
WHERE id = $1;

-- name: FindProductSKUsByProductID :many
SELECT id, product_id, external_sku_id, origin_sku_id, sku_name,
       color, size, price, original_price, currency,
       image_url, sku_properties, created_at, updated_at
FROM gugu.sku
WHERE product_id = $1
ORDER BY created_at;

-- name: FindProductSKUByProductIDAndExternalSKUID :one
SELECT id, product_id, external_sku_id, origin_sku_id, sku_name,
       color, size, price, original_price, currency,
       image_url, sku_properties, created_at, updated_at
FROM gugu.sku
WHERE product_id = $1 AND external_sku_id = $2;

-- name: CountSKUsByProductID :one
SELECT COUNT(*) FROM gugu.sku WHERE product_id = $1;
