package collectionmgr

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

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
		response.WriteError(c, http.StatusNotFound, err)
	case int(response.ErrCodeCollectionConflict), int(response.ErrCodeCollectionInvalid):
		response.WriteError(c, http.StatusBadRequest, err)
	case int(response.ErrCodeInternal):
		response.WriteError(c, http.StatusInternalServerError, err)
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
		response.WriteError(c, http.StatusInternalServerError, response.ErrInternal(err))
	}
}
