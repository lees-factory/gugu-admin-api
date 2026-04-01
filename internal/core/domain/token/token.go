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
