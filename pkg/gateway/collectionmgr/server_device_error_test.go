package collectionmgr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/i18n"
)

func TestWriteErrDeviceResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(i18n.Middleware())
	engine.PUT("/", func(c *gin.Context) {
		writeErr(c, &collection.DeviceError{
			Field:     "actions[0].name",
			Code:      "16",
			MessageID: "error.hostlink_end_code_16",
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/?lang=zh-CN", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	var result response.Result
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Equal(t, int(response.ErrCodeDeviceResponse), result.Code)
	assert.Equal(t, "设备返回错误。", result.Message)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, int(response.ErrCodeDeviceResponse), result.Errors[0].Code)
	assert.Equal(t, "actions[0].name", result.Errors[0].Field)
	assert.Equal(t, "设备结束码 16：不支持该写入命令。", result.Errors[0].Message)
}
