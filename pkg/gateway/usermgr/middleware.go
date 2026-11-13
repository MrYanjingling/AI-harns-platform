package usermgr

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"harns-platform/pkg/apis/response"
)

func (s *Service) RequireAuth() gin.HandlerFunc {
	if !s.authEnabled {
		return func(c *gin.Context) {
			c.Set(sessionContextKey, &Session{IsAdmin: true})
			c.Next()
		}
	}
	return func(c *gin.Context) {
		token := bearerToken(c.GetHeader("Authorization"))
		if token == "" {
			response.WriteError(c, http.StatusBadRequest, response.ErrLoginExpired())
			c.Abort()
			return
		}
		session, err := s.Authenticate(token)
		if err != nil {
			response.WriteError(c, http.StatusBadRequest, err)
			c.Abort()
			return
		}
		c.Set(sessionContextKey, session)
		c.Next()
	}
}

func (s *Service) RequireAdmin() gin.HandlerFunc {
	if !s.authEnabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}
	return func(c *gin.Context) {
		session := currentSession(c)
		if session == nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrLoginExpired())
			c.Abort()
			return
		}
		if !session.IsAdmin {
			response.WriteError(c, http.StatusBadRequest, response.ErrForbidden())
			c.Abort()
			return
		}
		c.Next()
	}
}

func currentSession(c *gin.Context) *Session {
	if c == nil {
		return nil
	}
	value, ok := c.Get(sessionContextKey)
	if !ok {
		return nil
	}
	session, _ := value.(*Session)
	return session
}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}
