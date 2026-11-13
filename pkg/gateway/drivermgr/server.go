package drivermgr

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/drivermgr/v1"
	"harns-platform/pkg/i18n"
)

// InstallHandler registers driver management routes.
func InstallHandler(group *gin.RouterGroup, svc *Service) {
	g := group.Group("/drivers")
	g.GET("", list(svc))
	g.POST("", create(svc))
	g.GET("/schema", getSchema(svc))
	g.GET("/:id", get(svc))
	g.PUT("/:id", update(svc))
	g.DELETE("/:id", deleteDriver(svc))
}

func list(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.DriverQuery
		if err := c.ShouldBindQuery(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		if req.Flat {
			items, err := svc.List(c.Request.Context(), req.DriverType)
			if err != nil {
				writeErr(c, err)
				return
			}
			response.WriteOK(c, &v1.DriverListVo{Drivers: items})
			return
		}
		groups, err := svc.ListGrouped(c.Request.Context(), req.DriverType)
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, &v1.DriverGroupListVo{Groups: groups})
	}
}

func getSchema(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.DriverSchemaQuery
		if err := c.ShouldBindQuery(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrRequestBody)
			return
		}
		item, err := svc.GetSchemaByModel(c.Request.Context(), req.DriverModel)
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, item)
	}
}

func create(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.CreateDriverReq
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		item, err := svc.Create(c.Request.Context(), &req)
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, item)
	}
}

func get(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		item, err := svc.Get(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, item)
	}
}

func update(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.UpdateDriverReq
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		item, err := svc.Update(c.Request.Context(), c.Param("id"), &req)
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, item)
	}
}

func deleteDriver(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, nil)
	}
}

func writeErr(c *gin.Context, err error) {
	var validation *ValidationError
	if errors.As(err, &validation) {
		lang := i18n.Lang(c)
		items := make([]response.ErrorItem, 0, len(validation.Issues))
		for _, issue := range validation.Issues {
			items = append(items, response.NewFieldItem(
				lang, response.ErrCodeDriverInvalid, issue.Field, issue.MsgID))
		}
		response.WriteFieldErrors(c, http.StatusBadRequest, response.ErrCodeDriverInvalid, items)
		return
	}
	switch response.ErrorCode(err) {
	case int(response.ErrCodeDriverNotFound):
		response.WriteError(c, http.StatusBadRequest, err)
	case int(response.ErrCodeDriverConflict), int(response.ErrCodeDriverBuiltinReadOnly):
		response.WriteError(c, http.StatusBadRequest, err)
	case int(response.ErrCodeInternal):
		response.WriteError(c, http.StatusBadRequest, err)
	default:
		response.WriteError(c, http.StatusBadRequest, err)
	}
}
