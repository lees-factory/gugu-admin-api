package batch

import (
	"reflect"
	"testing"
	"time"

	domaintoken "github.com/ljj/gugu-admin-api/internal/core/domain/token"
)

func TestPrioritizeAppType(t *testing.T) {
	tokens := []domaintoken.SellerToken{
		{AppType: domaintoken.AppTypeAffiliate, SellerID: "af"},
		{AppType: domaintoken.AppTypeDropshipping, SellerID: "ds"},
	}

	got := prioritizeAppType(tokens, domaintoken.AppTypeDropshipping)
	want := []domaintoken.SellerToken{
		{AppType: domaintoken.AppTypeDropshipping, SellerID: "ds"},
		{AppType: domaintoken.AppTypeAffiliate, SellerID: "af"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("prioritizeAppType() = %+v, want %+v", got, want)
	}
}

func TestRemoveAppType(t *testing.T) {
	tokens := []domaintoken.SellerToken{
		{AppType: domaintoken.AppTypeAffiliate, SellerID: "af"},
		{AppType: domaintoken.AppTypeDropshipping, SellerID: "ds"},
	}

	got := removeAppType(tokens, domaintoken.AppTypeDropshipping)
	want := []domaintoken.SellerToken{
		{AppType: domaintoken.AppTypeAffiliate, SellerID: "af"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("removeAppType() = %+v, want %+v", got, want)
	}
}

func TestShouldRefreshDropshippingToday(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	t.Run("same local day", func(t *testing.T) {
		now := time.Date(2026, time.April, 20, 12, 0, 0, 0, loc)
		last := time.Date(2026, time.April, 20, 0, 5, 0, 0, loc)
		if shouldRefreshDropshippingToday(now, last, loc) {
			t.Fatalf("shouldRefreshDropshippingToday() = true, want false")
		}
	})

	t.Run("new local day even within 24h", func(t *testing.T) {
		now := time.Date(2026, time.April, 20, 0, 1, 0, 0, loc)
		last := time.Date(2026, time.April, 19, 23, 59, 0, 0, loc)
		if !shouldRefreshDropshippingToday(now, last, loc) {
			t.Fatalf("shouldRefreshDropshippingToday() = false, want true")
		}
	})

	t.Run("zero last_refreshed_at", func(t *testing.T) {
		now := time.Date(2026, time.April, 20, 9, 0, 0, 0, loc)
		if !shouldRefreshDropshippingToday(now, time.Time{}, loc) {
			t.Fatalf("shouldRefreshDropshippingToday() = false, want true")
		}
	})
}
