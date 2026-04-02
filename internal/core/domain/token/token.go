package token

import "time"

type AppType string

const (
	AppTypeAffiliate    AppType = "AFFILIATE"
	AppTypeDropshipping AppType = "DROPSHIPPING"
)

type SellerToken struct {
	ID                    string
	SellerID              string
	HavanaID              string
	AppUserID             string
	UserNick              string
	Account               string
	AccountPlatform       string
	Locale                string
	SP                    string
	AccessToken           string
	RefreshToken          string
	AccessTokenExpiresAt  time.Time
	RefreshTokenExpiresAt *time.Time
	LastRefreshedAt       time.Time
	AuthorizedAt          time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
	AppType               AppType
}

func MergeRefreshedToken(existing SellerToken, refreshed SellerToken) SellerToken {
	refreshed.ID = existing.ID
	refreshed.SellerID = existing.SellerID
	refreshed.HavanaID = existing.HavanaID
	refreshed.AppUserID = existing.AppUserID
	refreshed.UserNick = existing.UserNick
	refreshed.Account = existing.Account
	refreshed.AccountPlatform = existing.AccountPlatform
	refreshed.Locale = existing.Locale
	refreshed.SP = existing.SP
	refreshed.AuthorizedAt = existing.AuthorizedAt
	refreshed.CreatedAt = existing.CreatedAt
	refreshed.AppType = existing.AppType
	return refreshed
}

func (t *SellerToken) IsAccessTokenExpired(now time.Time) bool {
	return now.After(t.AccessTokenExpiresAt)
}

func (t *SellerToken) IsRefreshTokenExpired(now time.Time) bool {
	if t.RefreshTokenExpiresAt == nil {
		return true
	}
	return now.After(*t.RefreshTokenExpiresAt)
}

func (t *SellerToken) AccessTokenExpiresSoon(now time.Time, margin time.Duration) bool {
	return now.Add(margin).After(t.AccessTokenExpiresAt)
}
