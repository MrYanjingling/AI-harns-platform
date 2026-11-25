package updatemgr

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"harns-platform/pkg/apis/response"
)

// InstallHandler registers the gateway system update endpoints.
func InstallHandler(group *gin.RouterGroup, svc *Service, admin gin.HandlerFunc) {
	if svc == nil {
		return
	}
	if admin == nil {
		admin = func(c *gin.Context) {
			response.WriteError(c, http.StatusBadRequest, response.ErrForbidden())
			c.Abort()
		}
	}
	group.GET("/system/update", getUpdate(svc))
	group.PUT("/system/update/settings", admin, updateSettings(svc))
	group.POST("/system/update/apply", admin, applyUpdate(svc))
	group.POST("/system/update/reject", admin, rejectUpdate(svc))
}

func getUpdate(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		response.WriteOK(c, svc.View())
	}
}

func updateSettings(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req UpdateSettingsReq
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		view, err := svc.UpdateSettings(c.Request.Context(), req)
		if err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrUpdateInvalid(err.Error()))
			return
		}
		response.WriteOK(c, view)
	}
}

func applyUpdate(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := svc.Apply(); err != nil {
			writeUpdateErr(c, err)
			return
		}
		response.WriteOK(c, svc.View())
	}
}

func rejectUpdate(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := svc.Reject(); err != nil {
			writeUpdateErr(c, err)
			return
		}
		response.WriteOK(c, svc.View())
	}
}

func writeUpdateErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errUpdateDisabled):
		response.WriteError(c, http.StatusBadRequest, response.ErrUpdateDisabled())
	case errors.Is(err, errUpdateNotFound):
		response.WriteError(c, http.StatusBadRequest, response.ErrUpdateTaskNotFound())
	case errors.Is(err, errUpdateBusy):
		response.WriteError(c, http.StatusBadRequest, response.ErrUpdateBusy())
	default:
		response.WriteError(c, http.StatusBadRequest, response.ErrUpdateFailed(err.Error()))
	}
}
