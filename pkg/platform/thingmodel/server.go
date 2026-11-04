package thingmodel

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/i18n"
	"harns-platform/pkg/platform/httpx"
	"harns-platform/pkg/platform/thingmodel/tsl"
	"harns-platform/pkg/platform/thingmodel/v1"
)

// InstallHandler registers the thing model routes on the given group.
func InstallHandler(group *gin.RouterGroup, svc *Service) {
	g := group.Group("/thingModels")
	g.POST("", httpx.WithOperation(httpx.OpCreate, "创建物模型"), create(svc))
	g.GET("", list(svc))
	g.GET("/:id", get(svc))
	g.PUT("/:id", httpx.WithOperation(httpx.OpUpdate, "更新物模型"), update(svc))
	g.DELETE("/:id", httpx.WithOperation(httpx.OpDelete, "删除物模型"), remove(svc))
	g.POST("/:id/properties", httpx.WithOperation(httpx.OpCreate, "追加物模型属性"), addProperties(svc))
	g.PUT("/:id/properties/:identifier", httpx.WithOperation(httpx.OpUpdate, "更新物模型属性"), updateProperty(svc))
	g.DELETE("/:id/properties", httpx.WithOperation(httpx.OpDelete, "删除物模型属性"), deleteProperties(svc))
	g.GET("/:id/tsl", exportTsl(svc))
	g.PUT("/:id/tsl", httpx.WithOperation(httpx.OpUpdate, "导入物模型"), importTsl(svc))
}

func create(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.CreateThingModelReq
		if !httpx.BindJSON(c, &req) {
			return
		}
		vo, err := svc.Create(c.Request.Context(), &req)
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, vo)
	}
}

func list(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.ThingModelQuery
		if !httpx.BindQuery(c, &req) {
			return
		}
		page, err := svc.List(c.Request.Context(), req)
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, page)
	}
}

func get(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		vo, err := svc.Get(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, vo)
	}
}

func update(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.UpdateThingModelReq
		if !httpx.BindJSON(c, &req) {
			return
		}
		vo, err := svc.Update(c.Request.Context(), c.Param("id"), &req)
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, vo)
	}
}

func remove(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		cascade, _ := strconv.ParseBool(c.Query("cascade"))
		if err := svc.Delete(c.Request.Context(), c.Param("id"), cascade); err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, nil)
	}
}

func addProperties(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var properties []tsl.Property
		if !httpx.BindJSON(c, &properties) {
			return
		}
		vo, err := svc.AddProperties(c.Request.Context(), c.Param("id"), properties)
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, vo)
	}
}

func updateProperty(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var property tsl.Property
		if !httpx.BindJSON(c, &property) {
			return
		}
		vo, err := svc.UpdateProperty(c.Request.Context(), c.Param("id"), c.Param("identifier"), property)
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, vo)
	}
}

func deleteProperties(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var identifiers []string
		if !httpx.BindJSON(c, &identifiers) {
			return
		}
		deleted, err := svc.DeleteProperties(c.Request.Context(), c.Param("id"), identifiers)
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, gin.H{"deleted": deleted})
	}
}

func exportTsl(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := svc.Export(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeErr(c, err)
			return
		}
		c.Data(http.StatusOK, "application/json; charset=utf-8", data)
	}
}

func importTsl(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		content, err := io.ReadAll(c.Request.Body)
		if err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrRequestBody)
			return
		}
		vo, err := svc.Import(c.Request.Context(), c.Param("id"), content)
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, vo)
	}
}

func writeErr(c *gin.Context, err error) {
	var validation *tsl.ValidationError
	if errors.As(err, &validation) {
		lang := i18n.Lang(c)
		items := make([]response.ErrorItem, 0, len(validation.Issues))
		for _, issue := range validation.Issues {
			items = append(items, response.NewFieldItem(
				lang, response.ErrCodeThingModelInvalid, issue.Field, issue.MsgID, issue.Args...))
		}
		response.WriteFieldErrors(c, http.StatusBadRequest, response.ErrCodeThingModelInvalid, items)
		return
	}
	response.WriteError(c, http.StatusBadRequest, err)
}
