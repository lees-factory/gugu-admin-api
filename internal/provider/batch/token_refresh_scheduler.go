package batch

import (
	"context"
	"log"
	"sync"
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
	affiliateRTMargin  time.Duration
	runMu              sync.Mutex
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

	go func() {
		// Operations policy: force-check DROPSHIPPING first at midnight (KST).
		startMidnightAlignedLoop(ctx, "token refresh scheduler (midnight ds-first)", 24*time.Hour, s.runMidnightDSFirst)
	}()
}

func (s *TokenRefreshScheduler) runOnce(ctx context.Context) {
	s.runMu.Lock()
	defer s.runMu.Unlock()

	tokens, err := s.tokenService.GetExpiringSoon(ctx, s.refreshMargin)
	if err != nil {
		log.Printf("token refresh scheduler: failed to get expiring tokens: %v", err)
		return
	}

	tokens = s.addDropshippingDailyCandidate(ctx, tokens)
	tokens = s.addAffiliateRefreshExpiryCandidate(ctx, tokens)
	tokens = prioritizeAppType(tokens, domaintoken.AppTypeDropshipping)

	if len(tokens) == 0 {
		log.Printf("token refresh scheduler: no tokens need refresh")
		return
	}

	for _, t := range tokens {
		s.refreshOne(ctx, t)
	}
}

func (s *TokenRefreshScheduler) runMidnightDSFirst(ctx context.Context) {
	s.runMu.Lock()
	defer s.runMu.Unlock()

	dsToken, err := s.tokenService.GetByAppType(ctx, domaintoken.AppTypeDropshipping)
	if err != nil {
		log.Printf("token refresh scheduler: failed to get dropshipping token for midnight run: %v", err)
	} else if dsToken == nil {
		log.Printf("token refresh scheduler: dropshipping token missing at midnight run")
	} else {
		now := s.tokenService.Now()
		if shouldRefreshDropshippingToday(now, dsToken.LastRefreshedAt, scheduleLocation) {
			log.Printf("token refresh scheduler: midnight ds-first refresh start (seller=%s)", dsToken.SellerID)
			// Force refresh attempt even when access token is not near expiry.
			s.refreshOne(ctx, *dsToken)
		} else {
			log.Printf("token refresh scheduler: midnight ds-first skipped; already refreshed today (seller=%s)", dsToken.SellerID)
		}
	}

	tokens, err := s.tokenService.GetExpiringSoon(ctx, s.refreshMargin)
	if err != nil {
		log.Printf("token refresh scheduler: failed to get expiring tokens after midnight ds-first run: %v", err)
		return
	}

	tokens = addAffiliateRefreshExpiryCandidateWithService(ctx, s.tokenService, s.affiliateRTMargin, tokens)
	tokens = removeAppType(tokens, domaintoken.AppTypeDropshipping)
	tokens = prioritizeAppType(tokens, domaintoken.AppTypeAffiliate)

	if len(tokens) == 0 {
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

	if !shouldRefreshDropshippingToday(s.tokenService.Now(), dsToken.LastRefreshedAt, scheduleLocation) {
		return tokens
	}

	return append(tokens, *dsToken)
}

func (s *TokenRefreshScheduler) addAffiliateRefreshExpiryCandidate(ctx context.Context, tokens []domaintoken.SellerToken) []domaintoken.SellerToken {
	return addAffiliateRefreshExpiryCandidateWithService(ctx, s.tokenService, s.affiliateRTMargin, tokens)
}

func addAffiliateRefreshExpiryCandidateWithService(
	ctx context.Context,
	tokenService *domaintoken.Service,
	affiliateRTMargin time.Duration,
	tokens []domaintoken.SellerToken,
) []domaintoken.SellerToken {
	if containsAppType(tokens, domaintoken.AppTypeAffiliate) {
		return tokens
	}

	afToken, err := tokenService.GetByAppType(ctx, domaintoken.AppTypeAffiliate)
	if err != nil {
		log.Printf("token refresh scheduler: failed to get affiliate token for refresh-token expiry check: %v", err)
		return tokens
	}
	if afToken == nil || afToken.RefreshTokenExpiresAt == nil {
		return tokens
	}

	if tokenService.Now().Add(affiliateRTMargin).Before(*afToken.RefreshTokenExpiresAt) {
		return tokens
	}

	return append(tokens, *afToken)
}

func (s *TokenRefreshScheduler) refreshOne(ctx context.Context, t domaintoken.SellerToken) {
	now := time.Now()

	if shouldSkipRefreshByTokenExpiry(t, now) {
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

func prioritizeAppType(tokens []domaintoken.SellerToken, appType domaintoken.AppType) []domaintoken.SellerToken {
	idx := -1
	for i := range tokens {
		if tokens[i].AppType == appType {
			idx = i
			break
		}
	}
	if idx <= 0 {
		return tokens
	}

	result := make([]domaintoken.SellerToken, 0, len(tokens))
	result = append(result, tokens[idx])
	result = append(result, tokens[:idx]...)
	result = append(result, tokens[idx+1:]...)
	return result
}

func removeAppType(tokens []domaintoken.SellerToken, appType domaintoken.AppType) []domaintoken.SellerToken {
	if len(tokens) == 0 {
		return tokens
	}
	result := make([]domaintoken.SellerToken, 0, len(tokens))
	for _, token := range tokens {
		if token.AppType == appType {
			continue
		}
		result = append(result, token)
	}
	return result
}

func shouldRefreshDropshippingToday(now time.Time, lastRefreshedAt time.Time, loc *time.Location) bool {
	if lastRefreshedAt.IsZero() {
		return true
	}

	nowLocal := now.In(loc)
	lastLocal := lastRefreshedAt.In(loc)

	if nowLocal.Year() != lastLocal.Year() {
		return true
	}

	return nowLocal.YearDay() != lastLocal.YearDay()
}

func shouldSkipRefreshByTokenExpiry(t domaintoken.SellerToken, now time.Time) bool {
	// Dropshipping refresh responses can omit refresh-token TTL metadata.
	// Avoid hard-skipping by local expiry timestamp and let upstream decide.
	if t.AppType == domaintoken.AppTypeDropshipping {
		return false
	}

	if t.RefreshTokenExpiresAt == nil {
		return false
	}

	return now.After(*t.RefreshTokenExpiresAt)
}
