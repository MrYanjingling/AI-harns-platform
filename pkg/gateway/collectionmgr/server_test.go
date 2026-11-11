package collectionmgr

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectionCreateRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service, _ := newTestService(t)
	engine := gin.New()
	InstallHandler(engine.Group("/api/v1"), service)

	body := `{
		"driverType":"modbus",
		"driverModel":"modbusTcp",
		"name":"modbus-tcp-1",
		"collectorCycleMs":1000,
		"protocolConfig":{"location":"127.0.0.1:502","memoryLayout":"ABCD"},
		"variables":[{
			"name":"temperature","dataType":"float32","address":"1!40001","accessMode":"r"
		}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"code":200`)
	assert.Contains(t, rec.Body.String(), `"id":`)
	assert.Contains(t, rec.Body.String(), `"protocolConfig":{"location":"127.0.0.1:502"`)
	assert.NotContains(t, rec.Body.String(), `"variables":[`)
}
