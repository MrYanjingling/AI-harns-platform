package things

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/i18n"
	"harns-platform/pkg/platform/httpx"
	"harns-platform/pkg/platform/things/v1"
)

// InstallHandler registers the things routes on the given group.
func InstallHandler(group *gin.RouterGroup, svc *Service) {
	g := group.Group("/things")
	g.POST("", httpx.WithOperation(httpx.OpCreate, "批量创建设备"), create(svc))
	g.GET("", list(svc))
	g.POST("/timeseries", getThingTimeSeries(svc))
	g.GET("/:id/basic", getBasic(svc))
	g.GET("/:id/features", getFeatureValues(svc))
	g.GET("/:id/shadow", getThingShadowData(svc))
	g.PUT("/:id", httpx.WithOperation(httpx.OpUpdate, "更新设备"), update(svc))
	g.DELETE("", httpx.WithOperation(httpx.OpDelete, "批量删除设备"), remove(svc))
}

func create(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.CreateThingsReq
		if !httpx.BindJSON(c, &req) {
			return
		}
		items, err := svc.Create(c.Request.Context(), &req)
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, items)
	}
}

func list(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.ThingQuery
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

func getThingTimeSeries(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.ThingTimeSeriesReq
		if !httpx.BindJSON(c, &req) {
			return
		}
		vo, err := svc.GetThingTimeSeries(c.Request.Context(), req)
		if err != nil {
			if response.ErrorCode(err) == int(response.ErrCodeInternal) {
				response.WriteError(c, http.StatusInternalServerError, err)
				return
			}
			writeErr(c, err)
			return
		}
		response.WriteOK(c, vo)
	}
}

func getBasic(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		vo, err := svc.GetBasic(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, vo)
	}
}

func getFeatureValues(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		values, err := svc.GetFeatureValues(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, values)
	}
}

func getThingShadowData(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.ThingShadowQuery
		if !httpx.BindQuery(c, &req) {
			return
		}
		vo, err := svc.GetThingShadowData(c.Request.Context(), c.Param("id"), req)
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, vo)
	}
}

func update(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.UpdateThingReq
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
		var ids []string
		if !httpx.BindJSON(c, &ids) {
			return
		}
		cascade, _ := strconv.ParseBool(c.Query("cascade"))
		deleted, err := svc.Delete(c.Request.Context(), ids, cascade)
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, gin.H{"deleted": deleted})
	}
}

func writeErr(c *gin.Context, err error) {
	var validation *ValidationError
	if errors.As(err, &validation) {
		lang := i18n.Lang(c)
		items := make([]response.ErrorItem, 0, len(validation.Issues))
		for _, issue := range validation.Issues {
			items = append(items, response.NewFieldItem(
				lang, response.ErrCodeThingInvalid, issue.Field, issue.MsgID, issue.Args...))
		}
		response.WriteFieldErrors(c, http.StatusBadRequest, response.ErrCodeThingInvalid, items)
		return
	}
	response.WriteError(c, http.StatusBadRequest, err)
}
