package assistant

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/ai/tool"
	"harns-platform/pkg/gateway/usermgr"
)

// InstallHandler registers the authenticated assistant routes.
func InstallHandler(group *gin.RouterGroup, svc *Service, admin gin.HandlerFunc) {
	g := group.Group("/ai")
	g.POST("/chat", chat(svc))
	if admin != nil {
		g.GET("/tools", admin, listTools(svc))
	}
}

func chat(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ChatRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		session := usermgr.CurrentSession(c)
		if session == nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrLoginExpired())
			return
		}
		result, err := svc.Chat(c.Request.Context(), tool.Principal{
			UserID:   session.UserID,
			Username: session.Username,
			IsAdmin:  session.IsAdmin,
			Source:   "assistant",
		}, &req)
		if err != nil {
			response.WriteError(c, http.StatusBadRequest, err)
			return
		}
		response.WriteOK(c, result)
	}
}

func listTools(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		items := make([]gin.H, 0)
		if svc != nil && svc.tools != nil {
			for _, item := range svc.tools.List() {
				items = append(items, gin.H{
					"name":        item.Name(),
					"description": item.Description(),
					"risk":        item.Risk(),
					"inputSchema": item.InputSchema(),
				})
			}
		}
		response.WriteOK(c, gin.H{"tools": items})
	}
}
