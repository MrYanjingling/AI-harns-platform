package collectionmgr

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/i18n"
)

func TestCollectionValidationReturnsMultipleLocalizedErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service, _ := newTestService(t)
	engine := gin.New()
	engine.Use(i18n.Middleware())
	InstallHandler(engine.Group("/api/v1"), service)

	body := `{
		"driverType":"modbus","driverModel":"modbusTcp",
		"collectorCycleMs":1000,"location":"127.0.0.1:502","memoryLayout":"ABCD",
		"variables":[{"name":"","dataType":"float32","address":"holding:1","accessMode":"bad"}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "zh-CN")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"code":20007`)
	assert.Contains(t, rec.Body.String(), `"field":"name"`)
	assert.Contains(t, rec.Body.String(), `"field":"variables[0].name"`)
	assert.Contains(t, rec.Body.String(), `"field":"variables[0].accessMode"`)
	assert.Contains(t, rec.Body.String(), `"message":"采集器参数无效。"`)
	assert.Contains(t, rec.Body.String(), `"message":"name 不能为空。"`)
}

func TestCollectionNotFoundIsLocalized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service, _ := newTestService(t)
	engine := gin.New()
	engine.Use(i18n.Middleware())
	InstallHandler(engine.Group("/api/v1"), service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/collections/missing", nil)
	req.Header.Set("Accept-Language", "zh-CN")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"code":20005`)
	assert.Contains(t, rec.Body.String(), `采集器 [missing] 不存在。`)
}
