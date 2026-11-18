package collectionmgr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/apis/response"
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
		"collectorCycleMs":1000,"protocolConfig":{"location":"127.0.0.1:502","memoryLayout":"ABCD"},
		"variables":[{"name":"","dataType":"float32","address":"1!40001","accessMode":"bad"}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "zh-CN")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	var result response.Result
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	require.Len(t, result.Errors, 3)
	assert.Equal(t, int(response.ErrCodeCollectionInvalid), result.Code)
	assert.Contains(t, result.Message, "name 不能为空。")
	assert.Equal(t, result.Errors[0].Message, result.Message)
	assert.Equal(t, "name", result.Errors[0].Field)
	assert.Equal(t, "variables[0].name", result.Errors[1].Field)
	assert.Equal(t, "variables[0].accessMode", result.Errors[2].Field)
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

	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"code":20005`)
	assert.Contains(t, rec.Body.String(), `采集器 [missing] 不存在。`)
}
