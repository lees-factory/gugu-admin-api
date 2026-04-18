package middleware

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ljj/gugu-admin-api/internal/core/support/response"
)

// AdminAuth validates admin authentication.
// If configuredKeys is empty, any non-empty key is accepted for backward compatibility.
func AdminAuth(configuredKeys []string, tokenVerifier AccessTokenVerifier) gin.HandlerFunc {
	allowedKeys := compactNonEmpty(configuredKeys)

	return func(c *gin.Context) {
		if c.Request.URL != nil {
			switch c.Request.URL.Path {
			case "/v1/admin/auth/login", "/v1/aliexpress/oauth/callback/affiliate", "/v1/aliexpress/oauth/callback/dropshipping":
				c.Next()
				return
			}
		}

		authorization := strings.TrimSpace(c.GetHeader("Authorization"))
		adminID, byBearer, bearerAttempted := authenticateByBearerToken(c, tokenVerifier, authorization)
		if byBearer {
			c.Set("admin_id", adminID)
			c.Set("admin_auth_type", "BEARER")
			c.Next()
			return
		}

		apiKey := strings.TrimSpace(c.GetHeader("X-Admin-API-Key"))
		if apiKey == "" {
			if bearerAttempted {
				c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrorFromCode(
					"A2000", "유효하지 않은 관리자 토큰입니다",
				))
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrorFromCode(
				"A2000", "인증이 필요합니다",
			))
			return
		}
		if len(allowedKeys) > 0 && !slices.Contains(allowedKeys, apiKey) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrorFromCode(
				"A2000", "유효하지 않은 관리자 키입니다",
			))
			return
		}

		c.Set("admin_auth_type", "API_KEY")
		c.Next()
	}
}

type AccessTokenVerifier interface {
	Verify(token string, now time.Time) (adminID string, expiresAt time.Time, err error)
}

func authenticateByBearerToken(c *gin.Context, tokenVerifier AccessTokenVerifier, authorization string) (string, bool, bool) {
	bearerToken, hasBearer := extractBearerToken(authorization)
	if !hasBearer {
		return "", false, false
	}
	if tokenVerifier == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrorFromCode(
			"A2000", "유효하지 않은 관리자 토큰입니다",
		))
		return "", false, true
	}
	adminID, _, err := tokenVerifier.Verify(bearerToken, time.Now())
	if err != nil {
		// Bearer token could be stale while API key remains valid.
		// In this case we continue to API key auth path instead of hard-failing.
		return "", false, true
	}
	return adminID, true, true
}

func extractBearerToken(value string) (string, bool) {
	if value == "" {
		return "", false
	}
	parts := strings.Fields(value)
	if len(parts) != 2 {
		return "", false
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}
	return token, true
}

func compactNonEmpty(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}
