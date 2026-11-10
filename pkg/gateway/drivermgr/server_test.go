package drivermgr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/i18n"
)

func TestDriverRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service, _ := newTestService(t)
	engine := gin.New()
	InstallHandler(engine.Group("/api/v1"), service)

	createBody := `{"driverType":"modbus","model":"customTcp","pluginName":"custom_tcp","schema":{"type":"object"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/drivers", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var created struct {
		Code int `json:"code"`
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.Equal(t, 200, created.Code)
	assert.NotEmpty(t, created.Data.ID)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/drivers", nil)
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"groups"`)
	assert.Contains(t, rec.Body.String(), `"customTcp"`)
	assert.Contains(t, rec.Body.String(), `"modbusTcp"`)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/drivers?flat=true", nil)
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"drivers"`)
	assert.NotContains(t, rec.Body.String(), `"groups"`)
}

func TestDriverRoutesReturnLocalizedFieldErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service, _ := newTestService(t)
	engine := gin.New()
	engine.Use(i18n.Middleware())
	InstallHandler(engine.Group("/api/v1"), service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/drivers", strings.NewReader(`{"driverType":"modbus","model":"custom","pluginName":"bad/name"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "zh-CN")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"code":20004`)
	assert.Contains(t, rec.Body.String(), `"field":"pluginName"`)
	assert.Contains(t, rec.Body.String(), `"message":"pluginName 只能包含字母、数字、下划线和连字符。"`)
}
