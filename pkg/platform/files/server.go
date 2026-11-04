package files

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/platform/files/v1"
	"harns-platform/pkg/platform/httpx"
)

// InstallHandler registers the file upload endpoint.
func InstallHandler(group *gin.RouterGroup, svc *Service) {
	group.POST("/files", httpx.WithOperation(httpx.OpCreate, "上传图片"), upload(svc))
}

func upload(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrRequestBody)
			return
		}
		key, err := svc.UploadImage(c.Request.Context(), file)
		if err != nil {
			switch {
			case errors.Is(err, ErrFileTooLarge):
				response.WriteError(c, http.StatusBadRequest, response.ErrFileTooLarge(svc.maxBytes))
			case errors.Is(err, ErrFileTypeInvalid):
				response.WriteError(c, http.StatusBadRequest, response.ErrFileTypeInvalid())
			default:
				response.WriteError(c, http.StatusInternalServerError, response.ErrFileUploadFailed(err))
			}
			return
		}
		response.WriteOK(c, v1.FileVo{Key: key})
	}
}
