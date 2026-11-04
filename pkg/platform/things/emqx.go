package things

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"

	"harns-platform/pkg/logging"
	"harns-platform/pkg/platform/things/v1"
)

// AuthSecretHeader carries the shared secret required by the EMQX endpoints.
const AuthSecretHeader = "X-Things-Auth-Secret"

// InstallEMQXHandler registers the two EMQX-facing endpoints outside the
// platform auth/audit middleware chain.
func InstallEMQXHandler(group *gin.RouterGroup, svc *Service, sink MessageSink, secret string) {
	group.Use(emqxSecretMiddleware(secret))
	group.POST("/thingsAuth", deviceAuth(svc))
	group.POST("/thingsPubMessage", pubMessage(svc, sink))
}

func emqxSecretMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if secret != "" {
			provided := c.GetHeader(AuthSecretHeader)
			if subtle.ConstantTimeCompare([]byte(provided), []byte(secret)) != 1 {
				writeAuthDeny(c)
				return
			}
		}
		c.Next()
	}
}

func deviceAuth(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.DeviceAuthReq
		if err := c.ShouldBindJSON(&req); err != nil {
			writeAuthDeny(c)
			return
		}
		resp, err := svc.Authenticate(c.Request.Context(), &req)
		if err != nil {
			logging.ErrorS(err, "Device authentication failed", "clientid", req.ClientID, "username", req.Username)
			writeAuthDeny(c)
			return
		}
		if resp.Result == v1.AuthResultDeny {
			c.JSON(http.StatusUnauthorized, resp)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func pubMessage(svc *Service, sink MessageSink) gin.HandlerFunc {
	return func(c *gin.Context) {
		var event EmqxPubEventMessage
		if err := c.ShouldBindJSON(&event); err != nil {
			c.Status(http.StatusNoContent)
			return
		}
		msg, ok := BuildDeviceMessageFromEvent(&event)
		if !ok {
			logging.V(2).InfoS("Ignoring unsupported thing message",
				"deviceKey", event.DeviceKey(), "topic", event.Topic, "event", event.Event)
			c.Status(http.StatusNoContent)
			return
		}
		if svc != nil && msg.ThingID != "" {
			svc.TouchThingOnline(c.Request.Context(), msg.ThingID, msg.Timestamp)
		}
		if err := sink.Handle(c.Request.Context(), msg); err != nil {
			logging.ErrorS(err, "Failed to handle thing message", "deviceKey", msg.DeviceKey, "topic", msg.Topic)
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func writeAuthDeny(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, v1.DeviceAuthResp{
		Result:    v1.AuthResultDeny,
		Superuser: false,
	})
}
