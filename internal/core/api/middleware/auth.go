package middleware

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ljj/gugu-admin-api/internal/core/support/response"
)

// AdminAuth validates admin authentication.
// If configuredKeys is empty, any non-empty key is accepted for backward compatibility.
func AdminAuth(configuredKeys []string) gin.HandlerFunc {
	allowedKeys := compactNonEmpty(configuredKeys)

	return func(c *gin.Context) {
		if c.Request.URL != nil {
			switch c.Request.URL.Path {
			case "/v1/aliexpress/oauth/callback/affiliate", "/v1/aliexpress/oauth/callback/dropshipping":
				c.Next()
				return
			}
		}

		apiKey := strings.TrimSpace(c.GetHeader("X-Admin-API-Key"))
		if apiKey == "" {
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

		c.Next()
	}
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
