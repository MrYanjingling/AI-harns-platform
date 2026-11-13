package usermgr

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/usermgr/v1"
	"harns-platform/pkg/gateway/validation"
	"harns-platform/pkg/i18n"
)

func InstallPublicHandler(group *gin.RouterGroup, svc *Service) {
	group.POST("/auth/login", login(svc))
}

func InstallHandler(group *gin.RouterGroup, svc *Service) {
	group.POST("/auth/logout", logout(svc))

	users := group.Group("/users")
	users.POST("", createUser(svc))
	users.GET("", listUsers(svc))
	users.GET("/:id", getUser(svc))
	users.PUT("/:id", svc.RequireAdmin(), updateUser(svc))
	users.DELETE("/:id", svc.RequireAdmin(), deleteUser(svc))
	users.PUT("/:id/password", changePassword(svc))
	users.POST("/:id/password/reset", svc.RequireAdmin(), resetPassword(svc))
}

func login(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.LoginReq
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		item, err := svc.Login(c.Request.Context(), req.Username, req.Password)
		if err != nil {
			writeUserErr(c, err)
			return
		}
		response.WriteOK(c, item)
	}
}

func logout(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := currentSession(c)
		if session == nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrLoginExpired())
			return
		}
		svc.Logout(session.Token)
		response.WriteOK(c, nil)
	}
}

func createUser(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.CreateUserReq
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		item, err := svc.Create(c.Request.Context(), &req)
		if err != nil {
			writeUserErr(c, err)
			return
		}
		response.WriteOK(c, item)
	}
}

func listUsers(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.UserQuery
		if err := c.ShouldBindQuery(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrRequestBody)
			return
		}
		page, err := svc.List(c.Request.Context(), req.PageQuery, req.Keyword)
		if err != nil {
			writeUserErr(c, err)
			return
		}
		response.WriteOK(c, page)
	}
}

func getUser(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		item, err := svc.Get(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeUserErr(c, err)
			return
		}
		response.WriteOK(c, item)
	}
}

func updateUser(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.UpdateUserReq
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		item, err := svc.Update(c.Request.Context(), c.Param("id"), &req)
		if err != nil {
			writeUserErr(c, err)
			return
		}
		response.WriteOK(c, item)
	}
}

func deleteUser(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
			writeUserErr(c, err)
			return
		}
		response.WriteOK(c, gin.H{"deleted": true})
	}
}

func changePassword(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := currentSession(c)
		if session == nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrLoginExpired())
			return
		}
		var req v1.ChangePasswordReq
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		item, err := svc.ChangePassword(c.Request.Context(), c.Param("id"), session.UserID, req.OldPassword, req.NewPassword)
		if err != nil {
			writeUserErr(c, err)
			return
		}
		response.WriteOK(c, item)
	}
}

func resetPassword(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		item, err := svc.ResetPassword(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeUserErr(c, err)
			return
		}
		response.WriteOK(c, item)
	}
}

func writeUserErr(c *gin.Context, err error) {
	var validationErr *validation.ValidationError
	if errors.As(err, &validationErr) {
		lang := i18n.Lang(c)
		items := make([]response.ErrorItem, 0, len(validationErr.Issues))
		for _, issue := range validationErr.Issues {
			items = append(items, response.NewFieldItem(
				lang, response.ErrCodeUserInvalid, issue.Field, issue.MsgID, issue.Args...))
		}
		response.WriteFieldErrors(c, http.StatusBadRequest, response.ErrCodeUserInvalid, items)
		return
	}
	response.WriteError(c, http.StatusBadRequest, err)
}
