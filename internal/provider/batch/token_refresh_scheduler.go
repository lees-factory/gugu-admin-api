package batch

import (
	"context"
	"log"
	"time"

	"github.com/ljj/gugu-admin-api/internal/clients/aliexpress"
	domaintoken "github.com/ljj/gugu-admin-api/internal/core/domain/token"
)

type TokenRefreshScheduler struct {
	tokenService       *domaintoken.Service
	affiliateClient    *aliexpress.PlatformClient
	dropshippingClient *aliexpress.PlatformClient
	interval           time.Duration
	refreshMargin      time.Duration
	dailyRefreshWindow time.Duration
	affiliateRTMargin  time.Duration
}

func NewTokenRefreshScheduler(
	tokenService *domaintoken.Service,
	affiliateClient *aliexpress.PlatformClient,
	dropshippingClient *aliexpress.PlatformClient,
	interval time.Duration,
) *TokenRefreshScheduler {
	return &TokenRefreshScheduler{
		tokenService:       tokenService,
		affiliateClient:    affiliateClient,
		dropshippingClient: dropshippingClient,
		interval:           interval,
		refreshMargin:      6 * time.Hour,
		dailyRefreshWindow: 24 * time.Hour,
		affiliateRTMargin:  24 * time.Hour,
	}
}

func (s *TokenRefreshScheduler) Start(ctx context.Context) {
	if s == nil || s.tokenService == nil || s.interval <= 0 {
		return
	}

	go func() {
		s.runOnce(ctx)

		startTickerLoop(ctx, "token refresh scheduler", s.interval, s.runOnce)
	}()
}

func (s *TokenRefreshScheduler) runOnce(ctx context.Context) {
	tokens, err := s.tokenService.GetExpiringSoon(ctx, s.refreshMargin)
	if err != nil {
		log.Printf("token refresh scheduler: failed to get expiring tokens: %v", err)
		return
	}

	tokens = s.addDropshippingDailyCandidate(ctx, tokens)
	tokens = s.addAffiliateRefreshExpiryCandidate(ctx, tokens)

	if len(tokens) == 0 {
		log.Printf("token refresh scheduler: no tokens need refresh")
		return
	}

	for _, t := range tokens {
		s.refreshOne(ctx, t)
	}
}

func containsAppType(tokens []domaintoken.SellerToken, appType domaintoken.AppType) bool {
	for _, t := range tokens {
		if t.AppType == appType {
			return true
		}
	}
	return false
}

func (s *TokenRefreshScheduler) addDropshippingDailyCandidate(ctx context.Context, tokens []domaintoken.SellerToken) []domaintoken.SellerToken {
	if containsAppType(tokens, domaintoken.AppTypeDropshipping) {
		return tokens
	}

	dsToken, err := s.tokenService.GetByAppType(ctx, domaintoken.AppTypeDropshipping)
	if err != nil {
		log.Printf("token refresh scheduler: failed to get dropshipping token for daily refresh: %v", err)
		return tokens
	}
	if dsToken == nil {
		return tokens
	}

	if s.tokenService.Now().Sub(dsToken.LastRefreshedAt) < s.dailyRefreshWindow {
		return tokens
	}

	return append(tokens, *dsToken)
}

func (s *TokenRefreshScheduler) addAffiliateRefreshExpiryCandidate(ctx context.Context, tokens []domaintoken.SellerToken) []domaintoken.SellerToken {
	if containsAppType(tokens, domaintoken.AppTypeAffiliate) {
		return tokens
	}

	afToken, err := s.tokenService.GetByAppType(ctx, domaintoken.AppTypeAffiliate)
	if err != nil {
		log.Printf("token refresh scheduler: failed to get affiliate token for refresh-token expiry check: %v", err)
		return tokens
	}
	if afToken == nil || afToken.RefreshTokenExpiresAt == nil {
		return tokens
	}

	if s.tokenService.Now().Add(s.affiliateRTMargin).Before(*afToken.RefreshTokenExpiresAt) {
		return tokens
	}

	return append(tokens, *afToken)
}

func (s *TokenRefreshScheduler) refreshOne(ctx context.Context, t domaintoken.SellerToken) {
	now := time.Now()

	if t.RefreshTokenExpiresAt != nil && now.After(*t.RefreshTokenExpiresAt) {
		log.Printf("token refresh scheduler: refresh token expired for seller=%s app_type=%s — re-authorization required", t.SellerID, t.AppType)
		return
	}
	if t.RefreshToken == "" {
		log.Printf("token refresh scheduler: refresh token missing for seller=%s app_type=%s — re-authorization required", t.SellerID, t.AppType)
		return
	}

	client := s.clientForAppType(t.AppType)
	if client == nil {
		log.Printf("token refresh scheduler: no client configured for seller=%s app_type=%s", t.SellerID, t.AppType)
		return
	}

	resp, err := client.RefreshToken(ctx, t.RefreshToken)
	if err != nil {
		log.Printf("token refresh scheduler: refresh failed for seller=%s app_type=%s: %v", t.SellerID, t.AppType, err)
		return
	}

	updated := domaintoken.MergeRefreshedToken(t, resp.ToDomainToken(t.AppType))

	if err := s.tokenService.SaveToken(ctx, updated); err != nil {
		log.Printf("token refresh scheduler: save failed for seller=%s app_type=%s: %v", t.SellerID, t.AppType, err)
		return
	}

	log.Printf("token refresh scheduler: refreshed token for seller=%s app_type=%s expires_at=%s",
		updated.SellerID, updated.AppType, updated.AccessTokenExpiresAt.Format(time.RFC3339))
}

func (s *TokenRefreshScheduler) clientForAppType(appType domaintoken.AppType) *aliexpress.PlatformClient {
	switch appType {
	case domaintoken.AppTypeAffiliate:
		return s.affiliateClient
	case domaintoken.AppTypeDropshipping:
		return s.dropshippingClient
	default:
		return nil
	}
}
