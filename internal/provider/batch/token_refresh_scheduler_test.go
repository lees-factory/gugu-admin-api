package batch

import (
	"reflect"
	"testing"

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
