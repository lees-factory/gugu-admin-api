package user

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	domainuser "github.com/ljj/gugu-admin-api/internal/core/domain/user"
	"github.com/ljj/gugu-admin-api/internal/core/support/response"
)

type Controller struct {
	userService *domainuser.Service
}

func NewController(userService *domainuser.Service) *Controller {
	return &Controller{userService: userService}
}

func (ctrl *Controller) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/users", ctrl.List)
	rg.GET("/users/:user_id/sessions", ctrl.ListSessions)
	rg.POST("/users/:user_id/sessions/revoke", ctrl.RevokeAllSessions)
	rg.POST("/users/:user_id/sessions/:session_id/revoke", ctrl.RevokeSessionByID)
	rg.POST("/users/:user_id/sessions/token-families/:token_family_id/revoke", ctrl.RevokeTokenFamily)
}

func (ctrl *Controller) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := ctrl.userService.List(c.Request.Context(), domainuser.ListFilter{
		Search:   c.Query("search"),
		Plan:     domainuser.Plan(c.Query("plan")),
		Status:   domainuser.Status(c.Query("status")),
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ErrorFromCode("USER_LIST_FAILED", err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"total_count": result.TotalCount,
		"page":        page,
		"page_size":   pageSize,
		"items":       toAdminUserListResponse(result.Users),
	}))
}

type adminUserResponse struct {
	ID               string                 `json:"id"`
	Email            string                 `json:"email"`
	DisplayName      string                 `json:"display_name"`
	Plan             string                 `json:"plan"`
	Status           string                 `json:"status"`
	EmailVerified    bool                   `json:"email_verified"`
	TrackedItemCount int64                  `json:"tracked_item_count"`
	CreatedAt        time.Time              `json:"created_at"`
	LastLoginAt      *time.Time             `json:"last_login_at"`
	Sessions         []adminSessionResponse `json:"sessions"`
}

type adminSessionResponse struct {
	ID               string     `json:"id"`
	UserID           string     `json:"user_id"`
	Status           string     `json:"status"`
	StatusReason     string     `json:"status_reason"`
	RefreshTokenHash string     `json:"refresh_token_hash"`
	TokenFamilyID    string     `json:"token_family_id"`
	ParentSessionID  *string    `json:"parent_session_id"`
	UserAgent        string     `json:"user_agent"`
	ClientIP         string     `json:"client_ip"`
	DeviceName       string     `json:"device_name"`
	ExpiresAt        time.Time  `json:"expires_at"`
	LastSeenAt       time.Time  `json:"last_seen_at"`
	RotatedAt        *time.Time `json:"rotated_at"`
	RevokedAt        *time.Time `json:"revoked_at"`
	ReuseDetectedAt  *time.Time `json:"reuse_detected_at"`
	CreatedAt        time.Time  `json:"created_at"`
}

func toAdminUserListResponse(users []domainuser.User) []adminUserResponse {
	result := make([]adminUserResponse, len(users))
	for i, user := range users {
		result[i] = adminUserResponse{
			ID:               user.ID,
			Email:            user.Email,
			DisplayName:      user.DisplayName,
			Plan:             string(user.Plan),
			Status:           string(user.Status),
			EmailVerified:    user.EmailVerified,
			TrackedItemCount: user.TrackedItemCount,
			CreatedAt:        user.CreatedAt,
			LastLoginAt:      user.LastLoginAt,
			Sessions:         toAdminSessionListResponse(user.Sessions),
		}
	}
	return result
}

func toAdminSessionListResponse(sessions []domainuser.LoginSession) []adminSessionResponse {
	result := make([]adminSessionResponse, len(sessions))
	now := time.Now()
	for i, session := range sessions {
		statusReason := session.StatusReason(now)
		result[i] = adminSessionResponse{
			ID:               session.ID,
			UserID:           session.UserID,
			Status:           string(session.Status(now)),
			StatusReason:     string(statusReason),
			RefreshTokenHash: session.RefreshTokenHash,
			TokenFamilyID:    session.TokenFamilyID,
			ParentSessionID:  session.ParentSessionID,
			UserAgent:        session.UserAgent,
			ClientIP:         session.ClientIP,
			DeviceName:       session.DeviceName,
			ExpiresAt:        session.ExpiresAt,
			LastSeenAt:       session.LastSeenAt,
			RotatedAt:        session.RotatedAt,
			RevokedAt:        session.RevokedAt,
			ReuseDetectedAt:  session.ReuseDetectedAt,
			CreatedAt:        session.CreatedAt,
		}
	}
	return result
}

func (ctrl *Controller) ListSessions(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, response.ErrorFromCode("INVALID_USER_ID", "user_id is required"))
		return
	}

	filter := domainuser.SessionListFilter{
		UserID: userID,
	}

	if statusRaw := c.Query("status"); statusRaw != "" {
		status := domainuser.Status(statusRaw)
		if status != domainuser.StatusActive && status != domainuser.StatusInactive {
			c.JSON(http.StatusBadRequest, response.ErrorFromCode("INVALID_STATUS", "status must be ACTIVE or INACTIVE"))
			return
		}
		filter.Status = status
	}

	if revokedRaw := c.Query("revoked"); revokedRaw != "" {
		revoked, err := strconv.ParseBool(revokedRaw)
		if err != nil {
			c.JSON(http.StatusBadRequest, response.ErrorFromCode("INVALID_REVOKED_FILTER", "revoked must be true or false"))
			return
		}
		filter.Revoked = &revoked
	}

	if reuseDetectedRaw := c.Query("reuse_detected"); reuseDetectedRaw != "" {
		reuseDetected, err := strconv.ParseBool(reuseDetectedRaw)
		if err != nil {
			c.JSON(http.StatusBadRequest, response.ErrorFromCode("INVALID_REUSE_DETECTED_FILTER", "reuse_detected must be true or false"))
			return
		}
		filter.ReuseDetected = &reuseDetected
	}

	sessions, err := ctrl.userService.ListSessions(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ErrorFromCode("USER_SESSION_LIST_FAILED", err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"user_id": userID,
		"count":   len(sessions),
		"items":   toAdminSessionListResponse(sessions),
	}))
}

func (ctrl *Controller) RevokeAllSessions(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, response.ErrorFromCode("INVALID_USER_ID", "user_id is required"))
		return
	}

	revokedCount, err := ctrl.userService.RevokeAllSessions(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ErrorFromCode("USER_SESSION_REVOKE_ALL_FAILED", err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"user_id":        userID,
		"revoked_count":  revokedCount,
		"operation_type": "REVOKE_ALL_BY_USER",
	}))
}

func (ctrl *Controller) RevokeSessionByID(c *gin.Context) {
	userID := c.Param("user_id")
	sessionID := c.Param("session_id")
	if userID == "" || sessionID == "" {
		c.JSON(http.StatusBadRequest, response.ErrorFromCode("INVALID_SESSION_TARGET", "user_id and session_id are required"))
		return
	}

	revoked, err := ctrl.userService.RevokeSessionByID(c.Request.Context(), userID, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ErrorFromCode("USER_SESSION_REVOKE_ONE_FAILED", err.Error()))
		return
	}
	if !revoked {
		c.JSON(http.StatusNotFound, response.ErrorFromCode("SESSION_NOT_FOUND", "session not found or already revoked"))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"user_id":        userID,
		"session_id":     sessionID,
		"revoked":        true,
		"operation_type": "REVOKE_ONE_BY_SESSION_ID",
	}))
}

func (ctrl *Controller) RevokeTokenFamily(c *gin.Context) {
	userID := c.Param("user_id")
	tokenFamilyID := c.Param("token_family_id")
	if userID == "" || tokenFamilyID == "" {
		c.JSON(http.StatusBadRequest, response.ErrorFromCode("INVALID_TOKEN_FAMILY_TARGET", "user_id and token_family_id are required"))
		return
	}

	revokedCount, err := ctrl.userService.RevokeTokenFamily(c.Request.Context(), userID, tokenFamilyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ErrorFromCode("USER_SESSION_REVOKE_FAMILY_FAILED", err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"user_id":         userID,
		"token_family_id": tokenFamilyID,
		"revoked_count":   revokedCount,
		"operation_type":  "REVOKE_BY_TOKEN_FAMILY",
	}))
}
