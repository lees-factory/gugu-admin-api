package adminauth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	domainadminauth "github.com/ljj/gugu-admin-api/internal/core/domain/adminauth"
	"github.com/ljj/gugu-admin-api/internal/core/support/response"
)

type Controller struct {
	authService *domainadminauth.Service
}

func NewController(authService *domainadminauth.Service) *Controller {
	return &Controller{authService: authService}
}

func (ctrl *Controller) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/admin/auth/login", ctrl.Login)
}

type loginRequest struct {
	ID       string `json:"id" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (ctrl *Controller) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorFromCode("INVALID_REQUEST", err.Error()))
		return
	}

	result, err := ctrl.authService.Login(c.Request.Context(), req.ID, req.Password)
	if err != nil {
		if errors.Is(err, domainadminauth.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, response.ErrorFromCode("INVALID_CREDENTIALS", "id 또는 password가 올바르지 않습니다"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.ErrorFromCode("ADMIN_LOGIN_FAILED", err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"admin_id":     result.AdminID,
		"login_id":     result.LoginID,
		"access_token": result.AccessToken,
		"token_type":   result.TokenType,
		"expires_at":   result.ExpiresAt,
	}))
}
