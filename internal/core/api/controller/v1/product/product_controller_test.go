package product

import (
	"encoding/json"
	"strings"
	"testing"

	domainproduct "github.com/ljj/gugu-admin-api/internal/core/domain/product"
	"github.com/ljj/gugu-admin-api/internal/core/enum"
)

func TestToProductListResponse_HidesPriceFields(t *testing.T) {
	items := []domainproduct.LocalizedProduct{
		{
			ID:                "p-1",
			Market:            enum.MarketAliExpress,
			ExternalProductID: "123456",
			OriginalURL:       "https://example.com/original",
			Title:             "sample",
			MainImageURL:      "https://example.com/image.jpg",
			ProductURL:        "https://example.com/product",
			CollectionSource:  domainproduct.CollectionSourceHotProductQuery,
			Language:          "KO",
		},
	}

	resp := toProductListResponse(items)
	if len(resp) != 1 {
		t.Fatalf("len(toProductListResponse()) = %d, want 1", len(resp))
	}

	encoded, err := json.Marshal(resp[0])
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	body := string(encoded)
	if strings.Contains(body, "\"current_price\"") {
		t.Fatalf("response must not include current_price: %s", body)
	}
	if strings.Contains(body, "\"currency\"") {
		t.Fatalf("response must not include currency: %s", body)
	}
}
