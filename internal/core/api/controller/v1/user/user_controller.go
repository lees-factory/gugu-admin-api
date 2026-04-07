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
