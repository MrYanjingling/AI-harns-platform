package aimgr

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/aimgr/v1"
	"harns-platform/pkg/gateway/validation"
	"harns-platform/pkg/i18n"
)

// InstallHandler registers admin-only AI configuration routes.
func InstallHandler(group *gin.RouterGroup, svc *Service, admin gin.HandlerFunc) {
	if admin == nil {
		admin = func(c *gin.Context) {
			response.WriteError(c, http.StatusBadRequest, response.ErrForbidden())
			c.Abort()
		}
	}
	g := group.Group("/ai")
	g.Use(admin)
	g.GET("/settings", getSettings(svc))
	g.PUT("/settings", updateSettings(svc))
	g.PUT("/model", updateModel(svc))
	g.POST("/model/test", testModel(svc))
}

func getSettings(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		item, err := svc.GetSettings(c.Request.Context())
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, item)
	}
}

func updateSettings(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.UpdateSettingsReq
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		item, err := svc.UpdateSettings(c.Request.Context(), &req)
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, item)
	}
}

func updateModel(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.UpdateModelReq
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		item, err := svc.UpdateModel(c.Request.Context(), &req)
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, item)
	}
}

func testModel(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.UpdateModelReq
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		if err := svc.TestModel(c.Request.Context(), &req); err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, gin.H{"success": true})
	}
}

func writeErr(c *gin.Context, err error) {
	var validationErr *validation.ValidationError
	if errors.As(err, &validationErr) {
		lang := i18n.Lang(c)
		items := make([]response.ErrorItem, 0, len(validationErr.Issues))
		for _, issue := range validationErr.Issues {
			items = append(items, response.NewFieldItem(
				lang, response.ErrCodeAIProviderInvalid, issue.Field, issue.MsgID, issue.Args...))
		}
		response.WriteFieldErrors(c, http.StatusBadRequest, response.ErrCodeAIProviderInvalid, items)
		return
	}
	switch response.ErrorCode(err) {
	case int(response.ErrCodeAIProviderInvalid):
		response.WriteError(c, http.StatusBadRequest, err)
	default:
		response.WriteError(c, http.StatusBadRequest, response.ErrInternal(err))
	}
}
