package pushmgr

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"harns-platform/pkg/apis"
	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/pushmgr/v1"
	"harns-platform/pkg/gateway/validation"
	"harns-platform/pkg/i18n"
)

type listQuery struct {
	apis.PageQuery
	Keyword string `form:"keyword"`
}

func InstallHandler(group *gin.RouterGroup, svc *Service) {
	g := group.Group("/push")
	g.POST("/connectors", createConnector(svc))
	g.GET("/connectors", listConnectors(svc))
	g.GET("/connectors/:id", getConnector(svc))
	g.PUT("/connectors/:id", updateConnector(svc))
	g.DELETE("/connectors/:id", deleteConnector(svc))
	g.POST("/groups", createGroup(svc))
	g.GET("/groups", listGroups(svc))
	g.GET("/groups/:id", getGroup(svc))
	g.PUT("/groups/:id", updateGroup(svc))
	g.DELETE("/groups/:id", deleteGroup(svc))
}

func createConnector(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.CreateConnectorReq
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		item, err := svc.CreateConnector(c.Request.Context(), &req)
		if err != nil {
			writeErr(c, err, response.ErrCodePushConnectorInvalid)
			return
		}
		response.WriteOK(c, item)
	}
}

func listConnectors(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req listQuery
		if err := c.ShouldBindQuery(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrRequestBody)
			return
		}
		page, err := svc.ListConnectors(c.Request.Context(), req.PageQuery, req.Keyword)
		if err != nil {
			writeErr(c, err, response.ErrCodePushConnectorInvalid)
			return
		}
		response.WriteOK(c, page)
	}
}

func getConnector(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		item, err := svc.GetConnector(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeErr(c, err, response.ErrCodePushConnectorInvalid)
			return
		}
		response.WriteOK(c, item)
	}
}

func updateConnector(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.UpdateConnectorReq
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		item, err := svc.UpdateConnector(c.Request.Context(), c.Param("id"), &req)
		if err != nil {
			writeErr(c, err, response.ErrCodePushConnectorInvalid)
			return
		}
		response.WriteOK(c, item)
	}
}

func deleteConnector(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		cascade, _ := strconv.ParseBool(c.Query("cascade"))
		if err := svc.DeleteConnector(c.Request.Context(), c.Param("id"), cascade); err != nil {
			writeErr(c, err, response.ErrCodePushConnectorInvalid)
			return
		}
		response.WriteOK(c, gin.H{"deleted": true})
	}
}

func createGroup(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.CreatePushGroupReq
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		item, err := svc.CreateGroup(c.Request.Context(), &req)
		if err != nil {
			writeErr(c, err, response.ErrCodePushGroupInvalid)
			return
		}
		response.WriteOK(c, item)
	}
}

func listGroups(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req listQuery
		if err := c.ShouldBindQuery(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrRequestBody)
			return
		}
		page, err := svc.ListGroups(c.Request.Context(), req.PageQuery, req.Keyword)
		if err != nil {
			writeErr(c, err, response.ErrCodePushGroupInvalid)
			return
		}
		response.WriteOK(c, page)
	}
}

func getGroup(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		item, err := svc.GetGroup(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeErr(c, err, response.ErrCodePushGroupInvalid)
			return
		}
		response.WriteOK(c, item)
	}
}

func updateGroup(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.UpdatePushGroupReq
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		item, err := svc.UpdateGroup(c.Request.Context(), c.Param("id"), &req)
		if err != nil {
			writeErr(c, err, response.ErrCodePushGroupInvalid)
			return
		}
		response.WriteOK(c, item)
	}
}

func deleteGroup(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := svc.DeleteGroup(c.Request.Context(), c.Param("id")); err != nil {
			writeErr(c, err, response.ErrCodePushGroupInvalid)
			return
		}
		response.WriteOK(c, gin.H{"deleted": true})
	}
}

func writeErr(c *gin.Context, err error, validationCode response.ErrCode) {
	var validationErr *validation.ValidationError
	if errors.As(err, &validationErr) {
		items := make([]response.ErrorItem, 0, len(validationErr.Issues))
		for _, issue := range validationErr.Issues {
			items = append(items, response.NewFieldItem(i18n.Lang(c), validationCode, issue.Field, issue.MsgID, issue.Args...))
		}
		response.WriteFieldErrors(c, http.StatusBadRequest, validationCode, items)
		return
	}
	switch {
	case errors.Is(err, ErrConnectorNotFound):
		response.WriteError(c, http.StatusNotFound, response.ErrPushConnectorNotFound(c.Param("id")))
	case errors.Is(err, ErrGroupNotFound):
		response.WriteError(c, http.StatusNotFound, response.ErrPushGroupNotFound(c.Param("id")))
	case errors.Is(err, ErrConnectorInUse):
		response.WriteError(c, http.StatusBadRequest, response.ErrPushConnectorConflict(c.Param("id")))
	default:
		response.WriteError(c, http.StatusInternalServerError, response.ErrInternal(err))
	}
}
