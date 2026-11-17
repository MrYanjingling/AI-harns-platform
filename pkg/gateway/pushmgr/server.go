package pushmgr

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"harns-platform/pkg/apis"
	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/pushmgr/v1"
	"harns-platform/pkg/gateway/validation"
	"harns-platform/pkg/i18n"
)

type listQuery struct {
	apis.PageQuery
	Keyword     string `form:"keyword"`
	Type        string `form:"type"`
	ConnectorID string `form:"connectorId"`
}

func InstallHandler(group *gin.RouterGroup, svc *Service) {
	g := group.Group("/push")
	g.POST("/connectors", createConnector(svc))
	g.GET("/connectors", listConnectors(svc))
	g.GET("/connectors/:id", getConnector(svc))
	g.PUT("/connectors/:id", updateConnector(svc))
	g.DELETE("/connectors/:id", deleteConnector(svc))
	g.POST("/connectors/:id/start", startConnector(svc))
	g.POST("/connectors/:id/stop", stopConnector(svc))
	g.POST("/connectors/:id/restart", restartConnector(svc))
	g.POST("/groups", createGroup(svc))
	g.GET("/groups", listGroups(svc))
	g.GET("/groups/:id", getGroup(svc))
	g.PUT("/groups/:id", updateGroup(svc))
	g.DELETE("/groups/:id", deleteGroup(svc))
	g.POST("/groups/:id/start", startGroup(svc))
	g.POST("/groups/:id/stop", stopGroup(svc))
	g.POST("/groups/:id/restart", restartGroup(svc))
	g.GET("/groups/:id/payloads/stream", streamGroupPayloads(svc))
	g.GET("/groups/:id/commands/stream", streamGroupCommands(svc))
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
		page, err := svc.ListConnectors(c.Request.Context(), req.PageQuery, req.Keyword, req.Type)
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

func startConnector(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		status, err := svc.StartConnector(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeErr(c, err, response.ErrCodePushConnectorInvalid)
			return
		}
		response.WriteOK(c, gin.H{"status": status})
	}
}

func stopConnector(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		status, err := svc.StopConnector(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeErr(c, err, response.ErrCodePushConnectorInvalid)
			return
		}
		response.WriteOK(c, gin.H{"status": status})
	}
}

func restartConnector(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		status, err := svc.RestartConnector(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeErr(c, err, response.ErrCodePushConnectorInvalid)
			return
		}
		response.WriteOK(c, gin.H{"status": status})
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
		page, err := svc.ListGroups(c.Request.Context(), req.PageQuery, req.Keyword, req.ConnectorID)
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

func startGroup(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		status, err := svc.StartGroup(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeErr(c, err, response.ErrCodePushGroupInvalid)
			return
		}
		response.WriteOK(c, gin.H{"status": status})
	}
}

func stopGroup(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		status, err := svc.StopGroup(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeErr(c, err, response.ErrCodePushGroupInvalid)
			return
		}
		response.WriteOK(c, gin.H{"status": status})
	}
}

func restartGroup(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		status, err := svc.RestartGroup(c.Param("id"))
		if err != nil {
			writeErr(c, err, response.ErrCodePushGroupInvalid)
			return
		}
		response.WriteOK(c, gin.H{"status": status})
	}
}

func streamGroupPayloads(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		ch, cancel, err := svc.SubscribePayloads(c.Param("id"))
		if err != nil {
			writeErr(c, err, response.ErrCodePushGroupInvalid)
			return
		}
		defer cancel()
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")
		if controller := http.NewResponseController(c.Writer); controller != nil {
			_ = controller.SetWriteDeadline(time.Time{})
		}
		c.Status(http.StatusOK)
		c.Writer.Flush()
		_, _ = c.Writer.WriteString("event: ready\ndata: {}\n\n")
		c.Writer.Flush()
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-c.Request.Context().Done():
				return
			case <-ticker.C:
				if _, err := c.Writer.WriteString(": heartbeat\n\n"); err != nil {
					return
				}
				c.Writer.Flush()
			case event, ok := <-ch:
				if !ok {
					return
				}
				data, err := json.Marshal(event)
				if err != nil {
					continue
				}
				if _, err := fmt.Fprintf(c.Writer, "event: payload\ndata: %s\n\n", data); err != nil {
					return
				}
				c.Writer.Flush()
			}
		}
	}
}

func streamGroupCommands(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		ch, cancel, err := svc.SubscribeCommands(c.Param("id"))
		if err != nil {
			writeErr(c, err, response.ErrCodePushGroupInvalid)
			return
		}
		defer cancel()
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")
		if controller := http.NewResponseController(c.Writer); controller != nil {
			_ = controller.SetWriteDeadline(time.Time{})
		}
		c.Status(http.StatusOK)
		c.Writer.Flush()
		_, _ = c.Writer.WriteString("event: ready\ndata: {}\n\n")
		c.Writer.Flush()
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-c.Request.Context().Done():
				return
			case <-ticker.C:
				if _, err := c.Writer.WriteString(": heartbeat\n\n"); err != nil {
					return
				}
				c.Writer.Flush()
			case event, ok := <-ch:
				if !ok {
					return
				}
				data, err := json.Marshal(event)
				if err != nil {
					continue
				}
				if _, err := fmt.Fprintf(c.Writer, "event: command\ndata: %s\n\n", data); err != nil {
					return
				}
				c.Writer.Flush()
			}
		}
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
	var startErr *ConnectorStartError
	if errors.As(err, &startErr) {
		response.WriteError(c, http.StatusBadRequest, response.ErrPushConnectorStartFailed(startErr.Reason))
		return
	}
	var stopErr *ConnectorStopError
	if errors.As(err, &stopErr) {
		response.WriteError(c, http.StatusBadRequest, response.ErrPushConnectorStopFailed(stopErr.Reason))
		return
	}
	switch {
	case errors.Is(err, ErrConnectorNotFound):
		response.WriteError(c, http.StatusBadRequest, response.ErrPushConnectorNotFound(c.Param("id")))
	case errors.Is(err, ErrGroupNotFound):
		response.WriteError(c, http.StatusBadRequest, response.ErrPushGroupNotFound(c.Param("id")))
	case errors.Is(err, ErrConnectorInUse):
		response.WriteError(c, http.StatusBadRequest, response.ErrPushConnectorConflict(c.Param("id")))
	case errors.Is(err, ErrConnectorAlreadyRunning):
		response.WriteError(c, http.StatusBadRequest, response.ErrPushConnectorAlreadyRunning(c.Param("id")))
	case errors.Is(err, ErrConnectorAlreadyStopped):
		response.WriteError(c, http.StatusBadRequest, response.ErrPushConnectorAlreadyStopped(c.Param("id")))
	case errors.Is(err, ErrConnectorStartFailed):
		response.WriteError(c, http.StatusBadRequest, response.ErrPushConnectorStartFailed(err.Error()))
	case errors.Is(err, ErrConnectorStopFailed):
		response.WriteError(c, http.StatusBadRequest, response.ErrPushConnectorStopFailed(err.Error()))
	case errors.Is(err, ErrGroupAlreadyRunning), errors.Is(err, ErrGroupAlreadyStopped),
		errors.Is(err, ErrGroupDisabled), errors.Is(err, ErrGroupConnectorNotRunning):
		response.WriteError(c, http.StatusBadRequest, response.ErrPushGroupInvalid())
	default:
		response.WriteError(c, http.StatusBadRequest, response.ErrInternal(err))
	}
}
