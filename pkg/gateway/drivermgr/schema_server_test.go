package drivermgr

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDriverSchemaByModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service, _ := newTestService(t)
	engine := gin.New()
	InstallHandler(engine.Group("/api/v1"), service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/drivers/schema?driverModel=modbusTcp", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"driverType":"modbus"`)
	assert.Contains(t, rec.Body.String(), `"driverModel":"modbusTcp"`)
	assert.Contains(t, rec.Body.String(), `"fields"`)
	assert.Contains(t, rec.Body.String(), `"location"`)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/drivers/schema?driverModel=missing", nil)
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())

	req = httptest.NewRequest(http.MethodGet, "/api/v1/drivers/schema", nil)
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}
