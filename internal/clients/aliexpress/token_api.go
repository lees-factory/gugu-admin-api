package aliexpress

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	domaintoken "github.com/ljj/gugu-admin-api/internal/core/domain/token"
)

type TokenResponse struct {
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	ExpiresIn             int64  `json:"expires_in"`
	RefreshExpiresIn      int64  `json:"refresh_expires_in"`
	ExpireTime            int64  `json:"expire_time"`
	RefreshTokenValidTime int64  `json:"refresh_token_valid_time"`
	SellerID              string `json:"seller_id"`
	UserID                string `json:"user_id"`
	HavanaID              string `json:"havana_id"`
	UserNick              string `json:"user_nick"`
	Account               string `json:"account"`
	AccountPlatform       string `json:"account_platform"`
	Locale                string `json:"locale"`
	SP                    string `json:"sp"`
}

func (c *PlatformClient) GenerateToken(ctx context.Context, code string) (*TokenResponse, error) {
	params := map[string]string{
		"code": code,
	}

	resp, err := c.CallSystemAPI(ctx, "/auth/token/create", params)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return parseTokenResponse(resp)
}

func (c *PlatformClient) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	params := map[string]string{
		"refresh_token": refreshToken,
	}

	resp, err := c.CallSystemAPI(ctx, "/auth/token/refresh", params)
	if err != nil {
		return nil, fmt.Errorf("refresh token: %w", err)
	}

	return parseTokenResponse(resp)
}

func parseTokenResponse(resp *PlatformResponse) (*TokenResponse, error) {
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("api error: code=%s type=%s message=%s", resp.Code, resp.Type, resp.Message)
	}

	var tokenResp TokenResponse

	if len(resp.Result) > 0 && string(resp.Result) != "null" {
		if err := json.Unmarshal(resp.Result, &tokenResp); err == nil && tokenResp.AccessToken != "" {
			return &tokenResp, nil
		}
	}

	if resp.RawBody != "" {
		if err := json.Unmarshal([]byte(resp.RawBody), &tokenResp); err == nil && tokenResp.AccessToken != "" {
			return &tokenResp, nil
		}

		var wrapped struct {
			Result TokenResponse `json:"result"`
		}
		if err := json.Unmarshal([]byte(resp.RawBody), &wrapped); err == nil && wrapped.Result.AccessToken != "" {
			return &wrapped.Result, nil
		}
	}

	return nil, fmt.Errorf("decode token response: unsupported structure")
}

// ToDomainToken converts a TokenResponse to a domain SellerToken.
func (r *TokenResponse) ToDomainToken(appType domaintoken.AppType) domaintoken.SellerToken {
	now := time.Now()

	accessExpiresAt := resolveExpireTime(r.ExpireTime, r.ExpiresIn, now)

	var refreshExpiresAt *time.Time
	if r.RefreshExpiresIn > 0 {
		t := resolveExpireTime(r.RefreshTokenValidTime, r.RefreshExpiresIn, now)
		refreshExpiresAt = &t
	}

	sellerID := r.SellerID
	if sellerID == "" {
		sellerID = r.UserID
	}

	return domaintoken.SellerToken{
		SellerID:              sellerID,
		HavanaID:              r.HavanaID,
		AppUserID:             r.UserID,
		UserNick:              r.UserNick,
		Account:               r.Account,
		AccountPlatform:       r.AccountPlatform,
		Locale:                r.Locale,
		SP:                    r.SP,
		AccessToken:           r.AccessToken,
		RefreshToken:          r.RefreshToken,
		AccessTokenExpiresAt:  accessExpiresAt,
		RefreshTokenExpiresAt: refreshExpiresAt,
		AuthorizedAt:          now,
		AppType:               appType,
	}
}

func resolveExpireTime(absoluteMillis int64, relativeSeconds int64, now time.Time) time.Time {
	if absoluteMillis > 0 {
		return time.UnixMilli(absoluteMillis)
	}
	if relativeSeconds > 0 {
		return now.Add(time.Duration(relativeSeconds) * time.Second)
	}
	return now
}

// ParseSellerID extracts seller_id from a raw token API JSON response string.
func ParseSellerID(raw string) string {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return ""
	}
	if v, ok := m["seller_id"]; ok {
		switch val := v.(type) {
		case string:
			return val
		case float64:
			return strconv.FormatInt(int64(val), 10)
		}
	}
	return ""
}
