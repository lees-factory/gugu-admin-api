-- name: InsertSKUPriceHistory :exec
INSERT INTO gugu.sku_price_history (
    sku_id, recorded_at, price, currency, change_value
) VALUES (
    $1, $2, $3, $4, $5
) ON CONFLICT (sku_id, currency, recorded_at) DO NOTHING;

-- name: GetLatestSKUPrice :one
SELECT sku_id, recorded_at, price, currency, change_value
FROM gugu.sku_price_history
WHERE sku_id = $1 AND currency = $2
ORDER BY recorded_at DESC
LIMIT 1;

-- name: UpsertSKUPriceSnapshot :exec
INSERT INTO gugu.sku_price_snapshot (
    sku_id, snapshot_date, price, original_price, currency
) VALUES (
    $1, $2, $3, $4, $5
) ON CONFLICT (sku_id, currency, snapshot_date) DO UPDATE SET
    price = EXCLUDED.price,
    original_price = EXCLUDED.original_price;
