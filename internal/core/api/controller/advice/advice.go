package advice

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	coreerror "github.com/ljj/gugu-admin-api/internal/core/error"
	"github.com/ljj/gugu-admin-api/internal/core/support/response"
)

// HandleError maps CoreException to HTTP response (= @ControllerAdvice)
func HandleError(c *gin.Context, err error) {
	var coreErr *coreerror.CoreException
	if errors.As(err, &coreErr) {
		status := mapKindToStatus(coreErr.Type.Kind)
		c.JSON(status, response.ErrorFromCode(coreErr.Type.Code, coreErr.Message))
		return
	}

	c.JSON(http.StatusInternalServerError, response.ErrorFromCode(
		coreerror.InternalError.Code,
		coreerror.InternalError.Message,
	))
}

func mapKindToStatus(kind coreerror.ErrorKind) int {
	switch kind {
	case coreerror.KindClient:
		return http.StatusBadRequest
	case coreerror.KindUnauthorized:
		return http.StatusUnauthorized
	case coreerror.KindServer:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
