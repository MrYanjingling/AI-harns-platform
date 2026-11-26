package assistant

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/ai/tool"
	"harns-platform/pkg/gateway/usermgr"
	"harns-platform/pkg/i18n"
)

// InstallHandler registers the authenticated assistant routes.
func InstallHandler(group *gin.RouterGroup, svc *Service, admin gin.HandlerFunc) {
	g := group.Group("/ai")
	g.POST("/chat", chat(svc))
	g.POST("/documents", uploadDocuments(svc))
	g.GET("/documents", listDocuments(svc))
	g.DELETE("/documents/:id", deleteDocument(svc))
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
		}, i18n.Lang(c), &req)
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

func uploadDocuments(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := usermgr.CurrentSession(c)
		if session == nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrLoginExpired())
			return
		}
		if err := c.Request.ParseMultipartForm(8 << 20); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrFileUploadFailed(err))
			return
		}
		if c.Request.MultipartForm != nil {
			defer func() { _ = c.Request.MultipartForm.RemoveAll() }()
		}
		files := c.Request.MultipartForm.File["files"]
		if len(files) == 0 {
			files = c.Request.MultipartForm.File["file"]
		}
		documents, err := svc.AddDocuments(c.Request.Context(), tool.Principal{
			UserID:   session.UserID,
			Username: session.Username,
			IsAdmin:  session.IsAdmin,
			Source:   "assistant",
		}, files)
		if err != nil {
			response.WriteError(c, http.StatusBadRequest, err)
			return
		}
		response.WriteOK(c, gin.H{"documents": documents})
	}
}

func listDocuments(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := usermgr.CurrentSession(c)
		if session == nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrLoginExpired())
			return
		}
		documents, err := svc.ListDocuments(c.Request.Context(), tool.Principal{
			UserID:   session.UserID,
			Username: session.Username,
			IsAdmin:  session.IsAdmin,
			Source:   "assistant",
		})
		if err != nil {
			response.WriteError(c, http.StatusBadRequest, err)
			return
		}
		response.WriteOK(c, gin.H{"documents": documents})
	}
}

func deleteDocument(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := usermgr.CurrentSession(c)
		if session == nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrLoginExpired())
			return
		}
		id := strings.TrimSpace(c.Param("id"))
		if id == "" {
			response.WriteError(c, http.StatusBadRequest, response.ErrResourceNotFound("document"))
			return
		}
		if err := svc.DeleteDocument(c.Request.Context(), tool.Principal{
			UserID:   session.UserID,
			Username: session.Username,
			IsAdmin:  session.IsAdmin,
			Source:   "assistant",
		}, id); err != nil {
			response.WriteError(c, http.StatusBadRequest, err)
			return
		}
		response.WriteOK(c, gin.H{})
	}
}
