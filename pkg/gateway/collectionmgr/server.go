package collectionmgr

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collectionmgr/v1"
	"harns-platform/pkg/gateway/validation"
	"harns-platform/pkg/i18n"
)

// InstallHandler registers collection routes.
func InstallHandler(group *gin.RouterGroup, svc *Service) {
	g := group.Group("/collections")
	g.POST("", create(svc))
	g.GET("", list(svc))
	g.GET("/:id", getDetail(svc))
	g.PUT("/:id", update(svc))
	g.DELETE("/:id", deleteCollection(svc))
	g.POST("/:id/variables", addVariables(svc))
	g.DELETE("/:id/variables", deleteVariables(svc))
	g.GET("/:id/variables", listVariables(svc))
	g.PUT("/:id/action", deliverAction(svc))
	g.POST("/:id/restart", restartCollection(svc))
	g.POST("/:id/stop", stopCollection(svc))
	g.POST("/:id/start", startCollection(svc))
	g.PUT("/:id/enable", setCollectionEnable(svc))
	g.GET("/:id/frames/requests/stream", streamFrames(svc, true))
	g.GET("/:id/frames/responses/stream", streamFrames(svc, false))
}

func create(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrRequestBody)
			return
		}
		item, err := svc.Create(c.Request.Context(), body)
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, item)
	}
}

func list(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.CollectionQuery
		if err := c.ShouldBindQuery(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrRequestBody)
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

func getDetail(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		item, err := svc.GetDetail(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, item)
	}
}

func update(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrRequestBody)
			return
		}
		item, err := svc.Update(c.Request.Context(), c.Param("id"), body)
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, item)
	}
}

func deleteCollection(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		deleted, err := svc.Delete(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, gin.H{"deleted": deleted})
	}
}

func addVariables(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var variables []*collection.Variable
		if err := c.ShouldBindJSON(&variables); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		result, err := svc.AddVariables(c.Request.Context(), c.Param("id"), variables)
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, result)
	}
}

func deleteVariables(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var names []string
		if err := c.ShouldBindJSON(&names); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		deleted, err := svc.DeleteVariables(c.Request.Context(), c.Param("id"), names)
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, gin.H{"deleted": deleted})
	}
}

func listVariables(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.VariableQuery
		if err := c.ShouldBindQuery(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrRequestBody)
			return
		}
		page, err := svc.ListVariables(c.Request.Context(), c.Param("id"), req)
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, page)
	}
}

func writeErr(c *gin.Context, err error) {
	var validationErr *validation.ValidationError
	if errors.As(err, &validationErr) {
		lang := i18n.Lang(c)
		items := make([]response.ErrorItem, 0, len(validationErr.Issues))
		for _, issue := range validationErr.Issues {
			items = append(items, response.NewFieldItem(
				lang, response.ErrCodeCollectionInvalid, issue.Field, issue.MsgID, issue.Args...))
		}
		response.WriteFieldErrors(c, http.StatusBadRequest, response.ErrCodeCollectionInvalid, items)
		return
	}

	switch response.ErrorCode(err) {
	case int(response.ErrCodeCollectionNotFound):
		response.WriteError(c, http.StatusBadRequest, err)
	case int(response.ErrCodeCollectionConflict), int(response.ErrCodeCollectionInvalid):
		response.WriteError(c, http.StatusBadRequest, err)
	case int(response.ErrCodeInternal):
		response.WriteError(c, http.StatusBadRequest, err)
	default:
		var syntaxErr *json.SyntaxError
		if errors.As(err, &syntaxErr) || errors.Is(err, io.ErrUnexpectedEOF) {
			response.WriteError(c, http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		if errors.Is(err, ErrUnsupportedDriver) {
			response.WriteError(c, http.StatusBadRequest, response.ErrCollectionInvalid())
			return
		}
		response.WriteError(c, http.StatusBadRequest, response.ErrInternal(err))
	}
}

func deliverAction(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.ActionReq
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		result, err := svc.DeliverAction(c.Request.Context(), c.Param("id"), req)
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, result)
	}
}

func streamFrames(svc *Service, requests bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ch, cancel, err := svc.SubscribeFrames(c.Request.Context(), c.Param("id"), requests)
		if err != nil {
			writeErr(c, err)
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
		event := "response"
		if requests {
			event = "request"
		}
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
			case frame, ok := <-ch:
				if !ok {
					return
				}
				payload, err := json.Marshal(frame)
				if err != nil {
					continue
				}
				if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, payload); err != nil {
					return
				}
				c.Writer.Flush()
			}
		}
	}
}

func restartCollection(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := svc.Restart(c.Request.Context(), c.Param("id")); err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, gin.H{"restarted": true})
	}
}

func stopCollection(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		status, err := svc.Stop(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, gin.H{"status": status})
	}
}

func startCollection(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		status, err := svc.Start(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, gin.H{"status": status})
	}
}

func setCollectionEnable(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.SetEnableReq
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		result, err := svc.SetEnable(c.Request.Context(), c.Param("id"), req.Enable)
		if err != nil {
			writeErr(c, err)
			return
		}
		response.WriteOK(c, result)
	}
}
