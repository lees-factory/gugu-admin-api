package aliexpress

import (
	"testing"
	"time"

	domaintoken "github.com/ljj/gugu-admin-api/internal/core/domain/token"
)

func TestParseTokenResponse_ParsesTopLevelTokenFields(t *testing.T) {
	resp := &PlatformResponse{
		RawBody: `{
			"access_token":"new-access",
			"refresh_token":"new-refresh",
			"expires_in":3600,
			"refresh_token_valid_time":1799999999000,
			"seller_id":"2457644580",
			"user_id":"2457644580",
			"havana_id":"133636586439",
			"user_nick":"kr861784585jyzae",
			"account":"wjdrk70@gmail.com",
			"account_platform":"buyerApp",
			"locale":"zh_CN",
			"sp":"ae"
		}`,
	}

	tokenResp, err := parseTokenResponse(resp)
	if err != nil {
		t.Fatalf("parseTokenResponse() error = %v", err)
	}
	if tokenResp.AccessToken != "new-access" {
		t.Fatalf("access_token = %q, want %q", tokenResp.AccessToken, "new-access")
	}
	if tokenResp.ExpiresIn != 3600 {
		t.Fatalf("expires_in = %d, want %d", tokenResp.ExpiresIn, 3600)
	}
	if tokenResp.SellerID != "2457644580" {
		t.Fatalf("seller_id = %q, want %q", tokenResp.SellerID, "2457644580")
	}
}

func TestToDomainToken_DropshippingFallbackRefreshExpiry(t *testing.T) {
	resp := &TokenResponse{
		AccessToken:  "new-access",
		RefreshToken: "new-refresh",
		ExpiresIn:    3600,
		// refresh_expires_in and refresh_token_valid_time are intentionally empty.
	}

	before := time.Now()
	domain := resp.ToDomainToken(domaintoken.AppTypeDropshipping)
	after := time.Now()

	if domain.RefreshTokenExpiresAt == nil {
		t.Fatalf("RefreshTokenExpiresAt must not be nil for dropshipping fallback")
	}

	min := before.Add(47 * time.Hour)
	max := after.Add(49 * time.Hour)
	if domain.RefreshTokenExpiresAt.Before(min) || domain.RefreshTokenExpiresAt.After(max) {
		t.Fatalf("RefreshTokenExpiresAt=%s out of expected fallback range [%s, %s]",
			domain.RefreshTokenExpiresAt.Format(time.RFC3339), min.Format(time.RFC3339), max.Format(time.RFC3339))
	}
}
