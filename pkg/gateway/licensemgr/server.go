package licensemgr

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/validation"
	"harns-platform/pkg/i18n"
)

// InstallHandler registers the license import and query endpoints.
func InstallHandler(group *gin.RouterGroup, svc *Service, admin gin.HandlerFunc) {
	if admin == nil {
		admin = func(c *gin.Context) {
			response.WriteError(c, http.StatusBadRequest, response.ErrForbidden())
			c.Abort()
		}
	}
	group.GET("/license", getLicense(svc))
	group.POST("/license/import", admin, importLicense(svc))
}

func getLicense(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		response.WriteOK(c, svc.GetStatus())
	}
}

func importLicense(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ImportRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		status, err := svc.Import(c.Request.Context(), req.Data)
		if err != nil {
			writeLicenseErr(c, err)
			return
		}
		response.WriteOK(c, status)
	}
}

func writeLicenseErr(c *gin.Context, err error) {
	var validationErr *validation.ValidationError
	if errors.As(err, &validationErr) {
		lang := i18n.Lang(c)
		items := make([]response.ErrorItem, 0, len(validationErr.Issues))
		for _, issue := range validationErr.Issues {
			items = append(items, response.NewFieldItem(
				lang, response.ErrCodeLicenseInvalid, issue.Field, issue.MsgID, issue.Args...))
		}
		response.WriteFieldErrors(c, http.StatusBadRequest, response.ErrCodeLicenseInvalid, items)
		return
	}
	switch {
	case errors.Is(err, ErrPublicKeyUnavailable):
		response.WriteError(c, http.StatusBadRequest, response.ErrLicensePublicKeyUnavailable())
	case errors.Is(err, ErrGatewayMismatch):
		response.WriteError(c, http.StatusBadRequest, response.ErrLicenseGatewayMismatch())
	case errors.Is(err, ErrExpired):
		response.WriteError(c, http.StatusBadRequest, response.ErrLicenseExpired())
	case errors.Is(err, ErrInvalid):
		response.WriteError(c, http.StatusBadRequest, response.ErrLicenseInvalid())
	default:
		response.WriteError(c, http.StatusBadRequest, response.ErrInternal(err))
	}
}
