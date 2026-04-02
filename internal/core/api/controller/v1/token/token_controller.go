package token

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ljj/gugu-admin-api/internal/clients/aliexpress"
	domaintoken "github.com/ljj/gugu-admin-api/internal/core/domain/token"
	"github.com/ljj/gugu-admin-api/internal/core/support/response"
)

type Controller struct {
	tokenService       *domaintoken.Service
	affiliateClient    *aliexpress.PlatformClient
	dropshippingClient *aliexpress.PlatformClient
}

func NewController(
	tokenService *domaintoken.Service,
	affiliateClient *aliexpress.PlatformClient,
	dropshippingClient *aliexpress.PlatformClient,
) *Controller {
	return &Controller{
		tokenService:       tokenService,
		affiliateClient:    affiliateClient,
		dropshippingClient: dropshippingClient,
	}
}

func (ctrl *Controller) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/aliexpress/token/generate", ctrl.Generate)
	rg.POST("/aliexpress/token/refresh", ctrl.Refresh)
	rg.GET("/aliexpress/token/status", ctrl.Status)
}

type generateRequest struct {
	Code    string `json:"code" binding:"required"`
	AppType string `json:"app_type" binding:"required"`
}

func (ctrl *Controller) Generate(c *gin.Context) {
	var req generateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorFromCode("INVALID_REQUEST", err.Error()))
		return
	}

	appType := domaintoken.AppType(req.AppType)
	client := ctrl.clientForAppType(appType)
	if client == nil {
		c.JSON(http.StatusBadRequest, response.ErrorFromCode("INVALID_APP_TYPE", "unsupported app_type: "+req.AppType))
		return
	}

	tokenResp, err := client.GenerateToken(c.Request.Context(), req.Code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ErrorFromCode("TOKEN_GENERATE_FAILED", err.Error()))
		return
	}

	domainToken := tokenResp.ToDomainToken(appType)
	if err := ctrl.tokenService.SaveToken(c.Request.Context(), domainToken); err != nil {
		c.JSON(http.StatusInternalServerError, response.ErrorFromCode("TOKEN_SAVE_FAILED", err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"seller_id":                domainToken.SellerID,
		"app_type":                 domainToken.AppType,
		"access_token_expires_at":  domainToken.AccessTokenExpiresAt,
		"refresh_token_expires_at": domainToken.RefreshTokenExpiresAt,
	}))
}

type refreshRequest struct {
	AppType string `json:"app_type" binding:"required"`
}

func (ctrl *Controller) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorFromCode("INVALID_REQUEST", err.Error()))
		return
	}

	appType := domaintoken.AppType(req.AppType)
	client := ctrl.clientForAppType(appType)
	if client == nil {
		c.JSON(http.StatusBadRequest, response.ErrorFromCode("INVALID_APP_TYPE", "unsupported app_type: "+req.AppType))
		return
	}

	existing, err := ctrl.tokenService.GetByAppType(c.Request.Context(), appType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ErrorFromCode("TOKEN_LOOKUP_FAILED", err.Error()))
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, response.ErrorFromCode("TOKEN_NOT_FOUND", "no token found for app_type: "+req.AppType))
		return
	}

	tokenResp, err := client.RefreshToken(c.Request.Context(), existing.RefreshToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ErrorFromCode("TOKEN_REFRESH_FAILED", err.Error()))
		return
	}

	updated := domaintoken.MergeRefreshedToken(*existing, tokenResp.ToDomainToken(appType))

	if err := ctrl.tokenService.SaveToken(c.Request.Context(), updated); err != nil {
		c.JSON(http.StatusInternalServerError, response.ErrorFromCode("TOKEN_SAVE_FAILED", err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"seller_id":                updated.SellerID,
		"app_type":                 updated.AppType,
		"access_token_expires_at":  updated.AccessTokenExpiresAt,
		"refresh_token_expires_at": updated.RefreshTokenExpiresAt,
	}))
}

func (ctrl *Controller) Status(c *gin.Context) {
	appType := c.Query("app_type")

	types := []domaintoken.AppType{domaintoken.AppTypeAffiliate, domaintoken.AppTypeDropshipping}
	if appType != "" {
		types = []domaintoken.AppType{domaintoken.AppType(appType)}
	}

	var statuses []gin.H
	for _, at := range types {
		t, err := ctrl.tokenService.GetByAppType(c.Request.Context(), at)
		if err != nil {
			c.JSON(http.StatusInternalServerError, response.ErrorFromCode("TOKEN_LOOKUP_FAILED", err.Error()))
			return
		}

		if t == nil {
			statuses = append(statuses, gin.H{
				"app_type": at,
				"status":   "NOT_FOUND",
			})
			continue
		}

		status := "ACTIVE"
		if t.IsAccessTokenExpired(ctrl.tokenService.Now()) {
			status = "EXPIRED"
		} else if t.AccessTokenExpiresSoon(ctrl.tokenService.Now(), 6*60*60*1e9) { // 6h
			status = "EXPIRING_SOON"
		}

		entry := gin.H{
			"app_type":                 t.AppType,
			"seller_id":                t.SellerID,
			"user_nick":                t.UserNick,
			"status":                   status,
			"access_token_expires_at":  t.AccessTokenExpiresAt,
			"refresh_token_expires_at": t.RefreshTokenExpiresAt,
			"last_refreshed_at":        t.LastRefreshedAt,
		}
		statuses = append(statuses, entry)
	}

	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"tokens": statuses,
	}))
}

func (ctrl *Controller) clientForAppType(appType domaintoken.AppType) *aliexpress.PlatformClient {
	switch appType {
	case domaintoken.AppTypeAffiliate:
		return ctrl.affiliateClient
	case domaintoken.AppTypeDropshipping:
		return ctrl.dropshippingClient
	default:
		return nil
	}
}
