package user

import (
	"net/http"
	"strconv"

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
		"items":       result.Users,
	}))
}
