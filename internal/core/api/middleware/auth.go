package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ljj/gugu-admin-api/internal/core/support/response"
)

// AdminAuth validates admin authentication
// TODO: 실제 인증 로직 구현 (JWT, API Key 등)
func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-Admin-API-Key")
		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrorFromCode(
				"A2000", "인증이 필요합니다",
			))
			return
		}

		c.Next()
	}
}
