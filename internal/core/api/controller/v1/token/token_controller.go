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
	AppType string `json:"app_type"`
}

func (ctrl *Controller) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorFromCode("INVALID_REQUEST", err.Error()))
		return
	}

	if req.AppType == "" {
		ctrl.refreshAll(c)
		return
	}

	appType := domaintoken.AppType(req.AppType)
	result, statusCode, ok := ctrl.refreshAppType(c, appType)
	if !ok {
		c.JSON(statusCode, response.ErrorFromCode("INVALID_APP_TYPE", "unsupported app_type: "+req.AppType))
		return
	}
	if result.Error != nil {
		c.JSON(statusCode, response.ErrorFromCode(result.Error.Code, result.Error.Message))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"seller_id":                result.SellerID,
		"app_type":                 result.AppType,
		"access_token_expires_at":  result.AccessTokenExpiresAt,
		"refresh_token_expires_at": result.RefreshTokenExpiresAt,
	}))
}

type refreshResult struct {
	AppType               domaintoken.AppType `json:"app_type"`
	SellerID              string              `json:"seller_id,omitempty"`
	AccessTokenExpiresAt  any                 `json:"access_token_expires_at,omitempty"`
	RefreshTokenExpiresAt any                 `json:"refresh_token_expires_at,omitempty"`
	Error                 *response.Error     `json:"error,omitempty"`
}

func (ctrl *Controller) refreshAll(c *gin.Context) {
	appTypes := []domaintoken.AppType{domaintoken.AppTypeAffiliate, domaintoken.AppTypeDropshipping}
	results := make([]refreshResult, 0, len(appTypes))
	for _, appType := range appTypes {
		result, _, _ := ctrl.refreshAppType(c, appType)
		results = append(results, result)
	}

	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"results": results,
	}))
}

func (ctrl *Controller) refreshAppType(c *gin.Context, appType domaintoken.AppType) (refreshResult, int, bool) {
	result := refreshResult{AppType: appType}

	client := ctrl.clientForAppType(appType)
	if client == nil {
		return result, http.StatusBadRequest, false
	}

	existing, err := ctrl.tokenService.GetByAppType(c.Request.Context(), appType)
	if err != nil {
		result.Error = &response.Error{Code: "TOKEN_LOOKUP_FAILED", Message: err.Error()}
		return result, http.StatusInternalServerError, true
	}
	if existing == nil {
		result.Error = &response.Error{Code: "TOKEN_NOT_FOUND", Message: "no token found for app_type: " + string(appType)}
		return result, http.StatusNotFound, true
	}
	result.SellerID = existing.SellerID
	if existing.RefreshToken == "" {
		result.Error = &response.Error{Code: "TOKEN_REFRESH_FAILED", Message: "refresh token missing; re-authorization required"}
		return result, http.StatusBadRequest, true
	}

	tokenResp, err := client.RefreshToken(c.Request.Context(), existing.RefreshToken)
	if err != nil {
		result.Error = &response.Error{Code: "TOKEN_REFRESH_FAILED", Message: err.Error()}
		return result, http.StatusInternalServerError, true
	}

	updated := domaintoken.MergeRefreshedToken(*existing, tokenResp.ToDomainToken(appType))

	if err := ctrl.tokenService.SaveToken(c.Request.Context(), updated); err != nil {
		result.Error = &response.Error{Code: "TOKEN_SAVE_FAILED", Message: err.Error()}
		return result, http.StatusInternalServerError, true
	}

	result.SellerID = updated.SellerID
	result.AccessTokenExpiresAt = updated.AccessTokenExpiresAt
	result.RefreshTokenExpiresAt = updated.RefreshTokenExpiresAt
	return result, http.StatusOK, true
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
