package things

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/oklog/ulid/v2"

	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/logging"
)

// TokenHeader carries the device token issued at creation time.
const TokenHeader = "X-Things-Token"

// InstallHTTPReportHandler registers the device-facing HTTP report endpoint.
// It uses the device token instead of the platform user session.
func InstallHTTPReportHandler(group *gin.RouterGroup, svc *Service, sink MessageSink) {
	group.POST("/thingsData", httpReport(svc, sink))
}

func httpReport(svc *Service, sink MessageSink) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.TrimSpace(c.GetHeader(TokenHeader))
		if token == "" {
			response.WriteError(c, http.StatusUnauthorized, response.ErrUnauthorized())
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrRequestBody)
			return
		}
		if len(body) == 0 || !json.Valid(body) {
			response.WriteError(c, http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		kind, ok := ResolveInboundKind(string(body))
		if !ok {
			var meta struct {
				Type string `json:"type"`
			}
			_ = json.Unmarshal(body, &meta)
			logging.V(2).InfoS("Invalid thing report type", "type", meta.Type)
			writeErr(c, NewValidationError([]FieldIssue{{
				Field: "type",
				MsgID: "validate.thing_message_type_invalid",
			}}))
			return
		}

		identity, err := svc.GetThingByToken(c.Request.Context(), token)
		if err != nil {
			status := http.StatusInternalServerError
			if response.ErrorCode(err) == int(response.ErrCodeUnauthorized) {
				status = http.StatusUnauthorized
			}
			response.WriteError(c, status, err)
			return
		}

		now := time.Now()
		svc.TouchThingOnline(c.Request.Context(), identity.ID, now.UnixMilli())

		msg := &DeviceMessage{
			Kind:          kind,
			Topic:         identity.UpTopic,
			DeviceKey:     identity.DeviceKey,
			ThingID:       identity.ID,
			ThingModelID:  identity.ThingModelID,
			ThingModelKey: identity.ThingModelKey,
			DeviceName:    identity.DeviceName,
			Payload:       string(body),
			Timestamp:     now.UnixMilli(),
			MessageID:     "http-" + ulid.Make().String(),
			Source:        "http",
		}
		if err := sink.Handle(c.Request.Context(), msg); err != nil {
			response.WriteError(c, http.StatusInternalServerError, response.ErrInternal(err))
			return
		}
		response.WriteOK(c, nil)
	}
}
