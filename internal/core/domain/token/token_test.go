package token

import (
	"testing"
	"time"
)

func TestMergeRefreshedToken_PreservesIdentityFields(t *testing.T) {
	createdAt := time.Date(2026, 3, 21, 5, 21, 39, 0, time.UTC)
	authorizedAt := time.Date(2026, 3, 21, 5, 21, 39, 0, time.UTC)
	refreshExpiresAt := time.Date(2026, 9, 17, 11, 36, 20, 0, time.UTC)

	existing := SellerToken{
		ID:              "row-id",
		SellerID:        "2457644580",
		HavanaID:        "133636586439",
		AppUserID:       "2457644580",
		UserNick:        "kr861784585jyzae",
		Account:         "wjdrk70@gmail.com",
		AccountPlatform: "buyerApp",
		Locale:          "zh_CN",
		SP:              "ae",
		AuthorizedAt:    authorizedAt,
		CreatedAt:       createdAt,
		AppType:         AppTypeAffiliate,
	}

	refreshed := SellerToken{
		SellerID:              "different-seller",
		HavanaID:              "different-havana",
		AppUserID:             "different-user",
		UserNick:              "different-nick",
		Account:               "different-account",
		AccountPlatform:       "different-platform",
		Locale:                "en_US",
		SP:                    "other",
		AccessToken:           "new-access",
		RefreshToken:          "new-refresh",
		AccessTokenExpiresAt:  time.Date(2026, 4, 2, 20, 0, 0, 0, time.UTC),
		RefreshTokenExpiresAt: &refreshExpiresAt,
		AppType:               AppTypeDropshipping,
	}

	merged := MergeRefreshedToken(existing, refreshed)

	if merged.ID != existing.ID || merged.SellerID != existing.SellerID || merged.AppType != existing.AppType {
		t.Fatalf("identity fields were not preserved: %+v", merged)
	}
	if merged.AccessToken != "new-access" || merged.RefreshToken != "new-refresh" {
		t.Fatalf("refreshed token values were not preserved: %+v", merged)
	}
	if merged.CreatedAt != createdAt || merged.AuthorizedAt != authorizedAt {
		t.Fatalf("timestamp identity fields were not preserved: %+v", merged)
	}
}
