package productalias

import (
	"context"
	"crypto/md5"
	"database/sql"
	"fmt"
	"strings"

	"github.com/ljj/gugu-admin-api/internal/core/enum"
)

type SQLRepository struct {
	db *sql.DB
}

func NewSQLRepository(db *sql.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

func (r *SQLRepository) FindProductIDByAlias(ctx context.Context, market enum.Market, aliasExternalProductID string) (string, error) {
	const query = `
SELECT product_id
FROM gugu.product_external_alias
WHERE market = $1
  AND alias_external_product_id = $2
LIMIT 1`

	marketValue := strings.TrimSpace(string(market))
	aliasValue := strings.TrimSpace(aliasExternalProductID)
	if marketValue == "" || aliasValue == "" {
		return "", nil
	}

	var productID string
	err := r.db.QueryRowContext(ctx, query, marketValue, aliasValue).Scan(&productID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(productID), nil
}

func (r *SQLRepository) UpsertViewAlias(ctx context.Context, market enum.Market, aliasExternalProductID, productID string) error {
	const query = `
INSERT INTO gugu.product_external_alias (
    id, market, alias_external_product_id, product_id, alias_type, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, 'VIEW', NOW(), NOW()
)
ON CONFLICT (market, alias_external_product_id)
DO UPDATE SET
    product_id = EXCLUDED.product_id,
    alias_type = EXCLUDED.alias_type,
    updated_at = NOW()`

	marketValue := strings.TrimSpace(string(market))
	aliasValue := strings.TrimSpace(aliasExternalProductID)
	productID = strings.TrimSpace(productID)
	if marketValue == "" || aliasValue == "" || productID == "" {
		return nil
	}

	aliasID := deterministicAliasID(marketValue, aliasValue, "VIEW")
	_, err := r.db.ExecContext(ctx, query, aliasID, marketValue, aliasValue, productID)
	return err
}

func deterministicAliasID(market, aliasExternalProductID, aliasType string) string {
	sum := md5.Sum([]byte(fmt.Sprintf("%s:%s:%s", market, aliasExternalProductID, aliasType)))
	return fmt.Sprintf("%x", sum[:])
}
