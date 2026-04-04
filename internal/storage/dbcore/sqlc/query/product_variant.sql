-- name: UpsertProductVariant :exec
INSERT INTO gugu.product_variant (
    product_id, language, currency, title, main_image_url, product_url,
    current_price, last_collected_at, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
ON CONFLICT (product_id, language, currency) DO UPDATE SET
    title = EXCLUDED.title,
    main_image_url = EXCLUDED.main_image_url,
    product_url = EXCLUDED.product_url,
    current_price = EXCLUDED.current_price,
    last_collected_at = EXCLUDED.last_collected_at,
    updated_at = EXCLUDED.updated_at;
