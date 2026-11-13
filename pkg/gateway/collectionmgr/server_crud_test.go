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
)

func newTestEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	service, _ := newTestService(t)
	engine := gin.New()
	InstallHandler(engine.Group("/api/v1"), service)
	return engine
}

func createTestCollection(t *testing.T, engine *gin.Engine, body string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var result struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	require.NotEmpty(t, result.Data.ID)
	return result.Data.ID
}

const collectionBody = `{
	"driverType":"modbus","driverModel":"modbusTcp","name":"modbus-tcp-1",
	"collectorCycleMs":1000,"protocolConfig":{"location":"127.0.0.1:502","memoryLayout":"ABCD"},
	"variables":[{"name":"temperature","dataType":"float32","address":"1!40001","accessMode":"r"}]
}`

func TestCollectionListUpdateDetailDelete(t *testing.T) {
	engine := newTestEngine(t)
	id := createTestCollection(t, engine, collectionBody)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/collections?page=1&pageSize=1", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"total":1`)
	assert.Contains(t, rec.Body.String(), `"list":[`)
	assert.NotContains(t, rec.Body.String(), `"variables"`)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/collections/"+id, nil)
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"location":"127.0.0.1:502"`)
	assert.NotContains(t, rec.Body.String(), `"variables"`)

	updateBody := `{"name":"updated","collectorCycleMs":2000,"protocolConfig":{"location":"127.0.0.2:503","memoryLayout":"DCBA"}}`
	req = httptest.NewRequest(http.MethodPut, "/api/v1/collections/"+id, strings.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"name":"updated"`)
	assert.Contains(t, rec.Body.String(), `"version":"2"`)

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/collections/"+id, nil)
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"deleted":1`)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/collections/"+id, nil)
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}

func TestCollectionVariableBatchAndPagination(t *testing.T) {
	engine := newTestEngine(t)
	id := createTestCollection(t, engine, collectionBody)

	addBody := `[
		{"name":"humidity","dataType":"float32","address":"1!40002","accessMode":"r"},
		{"name":"voltage","dataType":"float32","address":"1!40003","accessMode":"r"}
	]`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections/"+id+"/variables", strings.NewReader(addBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"added":2`)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/collections/"+id+"/variables?page=1&pageSize=1", nil)
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"total":3`)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &struct {
		Data struct {
			List []json.RawMessage `json:"list"`
		} `json:"data"`
	}{}))
	assert.Contains(t, rec.Body.String(), `"pageSize":1`)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/collections/"+id+"/variables?keyword=hum&page=1&pageSize=10", nil)
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"total":1`)
	assert.Contains(t, rec.Body.String(), `"humidity"`)

	deleteBody := `["temperature","missing"]`
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/collections/"+id+"/variables", strings.NewReader(deleteBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"deleted":1`)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/collections/"+id+"/variables?page=1&pageSize=10", nil)
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"total":2`)
}
