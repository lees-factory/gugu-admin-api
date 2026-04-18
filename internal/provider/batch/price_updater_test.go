package batch

import (
	"reflect"
	"testing"

	domainproduct "github.com/ljj/gugu-admin-api/internal/core/domain/product"
)

func TestCurrenciesForProduct(t *testing.T) {
	product := domainproduct.Product{ID: "p1"}

	testCases := []struct {
		name     string
		filter   PriceUpdateFilter
		expected []string
	}{
		{
			name:     "default to KRW",
			filter:   PriceUpdateFilter{},
			expected: []string{"KRW"},
		},
		{
			name: "USD only",
			filter: PriceUpdateFilter{
				Currencies: []string{"USD"},
			},
			expected: []string{"USD"},
		},
		{
			name: "KRW and USD",
			filter: PriceUpdateFilter{
				Currencies: []string{"KRW", "USD"},
			},
			expected: []string{"KRW", "USD"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := currenciesForProduct(product, tc.filter)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("currenciesForProduct() = %+v, want %+v", got, tc.expected)
			}
		})
	}
}

func TestNormalizeTargetGroup_DefaultsToAll(t *testing.T) {
	if got := normalizeTargetGroup(""); got != TargetGroupAll {
		t.Fatalf("normalizeTargetGroup(\"\") = %s, want %s", got, TargetGroupAll)
	}
	if got := normalizeTargetGroup("hot_products"); got != TargetGroupHotProducts {
		t.Fatalf("normalizeTargetGroup(hot_products) = %s, want %s", got, TargetGroupHotProducts)
	}
	if got := normalizeTargetGroup("tracked"); got != TargetGroupTracked {
		t.Fatalf("normalizeTargetGroup(tracked) = %s, want %s", got, TargetGroupTracked)
	}
}
