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
	}
}

func (s *TokenRefreshScheduler) Start(ctx context.Context) {
	if s == nil || s.tokenService == nil || s.interval <= 0 {
		return
	}

	ticker := time.NewTicker(s.interval)

	go func() {
		defer ticker.Stop()

		log.Printf("token refresh scheduler started: interval=%s margin=%s", s.interval, s.refreshMargin)

		s.runOnce(ctx)

		for {
			select {
			case <-ctx.Done():
				log.Printf("token refresh scheduler stopped: %v", ctx.Err())
				return
			case <-ticker.C:
				s.runOnce(ctx)
			}
		}
	}()
}

func (s *TokenRefreshScheduler) runOnce(ctx context.Context) {
	tokens, err := s.tokenService.GetExpiringSoon(ctx, s.refreshMargin)
	if err != nil {
		log.Printf("token refresh scheduler: failed to get expiring tokens: %v", err)
		return
	}

	if len(tokens) == 0 {
		log.Printf("token refresh scheduler: no tokens need refresh")
		return
	}

	for _, t := range tokens {
		s.refreshOne(ctx, t)
	}
}

func (s *TokenRefreshScheduler) refreshOne(ctx context.Context, t domaintoken.SellerToken) {
	now := time.Now()

	if t.RefreshTokenExpiresAt != nil && now.After(*t.RefreshTokenExpiresAt) {
		log.Printf("token refresh scheduler: refresh token expired for seller=%s app_type=%s — re-authorization required", t.SellerID, t.AppType)
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
