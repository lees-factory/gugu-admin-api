package aliexpress

import "testing"

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
