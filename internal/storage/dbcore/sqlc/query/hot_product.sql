-- name: InsertHotProduct :exec
INSERT INTO gugu.hot_product (
    id, external_product_id, title, image_url, product_url,
    promotion_link, sale_price, currency, collected_date, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) ON CONFLICT (external_product_id, collected_date) DO NOTHING;

-- name: ListHotProductsByDate :many
SELECT id, external_product_id, title, image_url, product_url,
       promotion_link, sale_price, currency, collected_date, created_at
FROM gugu.hot_product
WHERE collected_date = $1
ORDER BY created_at;

-- name: ListHotProductsLatest :many
SELECT id, external_product_id, title, image_url, product_url,
       promotion_link, sale_price, currency, collected_date, created_at
FROM gugu.hot_product
WHERE collected_date = (SELECT MAX(collected_date) FROM gugu.hot_product)
ORDER BY created_at;

-- name: DeleteHotProductsBefore :execrows
DELETE FROM gugu.hot_product
WHERE collected_date < $1;
