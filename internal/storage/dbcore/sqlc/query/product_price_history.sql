-- name: InsertProductPriceHistory :exec
INSERT INTO gugu.product_price_history (
    product_id, recorded_at, price, currency, change_value, sku_id
) VALUES (
    $1, $2, $3, $4, $5, $6
) ON CONFLICT (product_id, currency, recorded_at) DO NOTHING;

-- name: GetLatestProductPrice :one
SELECT product_id, recorded_at, price, currency, change_value, sku_id
FROM gugu.product_price_history
WHERE product_id = $1 AND currency = $2
ORDER BY recorded_at DESC
LIMIT 1;

-- name: UpsertProductPriceSnapshot :exec
INSERT INTO gugu.product_price_snapshot (
    product_id, snapshot_date, price, currency
) VALUES (
    $1, $2, $3, $4
) ON CONFLICT (product_id, currency, snapshot_date) DO UPDATE SET
    price = EXCLUDED.price;
