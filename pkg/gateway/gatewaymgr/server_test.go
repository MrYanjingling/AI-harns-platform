package gatewaymgr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/sqlite"
	"harns-platform/pkg/version"
)

func newGatewayTestManager(t *testing.T) (*Manager, *Store) {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	store := NewStore(db)
	require.NoError(t, store.Initialize(ctx))
	service := NewService(store)
	meta, err := service.LoadOrCreate(ctx)
	require.NoError(t, err)
	manager := NewGatewayManager(service, nil)
	manager.gatewayMeta = meta
	return manager, store
}

func TestManagerUpdateGatewayName(t *testing.T) {
	manager, store := newGatewayTestManager(t)
	ctx := context.Background()
	before, err := manager.GetGatewayMeta()
	require.NoError(t, err)

	vo, err := manager.UpdateGatewayName(ctx, "  plant gateway  ")
	require.NoError(t, err)
	assert.Equal(t, "plant gateway", vo.Name)
	assert.NotEqual(t, before.Version, vo.Version)
	assert.Equal(t, before.ID, vo.ID)
	assert.Equal(t, before.Secret, vo.Secret)
	assert.Equal(t, before.InitTime, vo.InitTime)
	assert.Equal(t, before.ModTime, vo.ModTime)

	stored, err := store.GetMeta(ctx)
	require.NoError(t, err)
	assert.Equal(t, "plant gateway", stored.Name)

	copied, err := manager.GetGatewayMeta()
	require.NoError(t, err)
	copied.Name = "mutated"
	current, err := manager.GetGatewayMeta()
	require.NoError(t, err)
	assert.Equal(t, "plant gateway", current.Name)
}

func TestSystemInfoHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager, _ := newGatewayTestManager(t)
	engine := gin.New()
	InstallHandler(engine.Group("/api/v1"), manager)

	rec := serveGateway(engine, http.MethodGet, "/api/v1/systemInfo", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var result response.Result
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	data := result.Data.(map[string]interface{})
	assert.Equal(t, version.Version(), data["version"])
	assert.Equal(t, version.CommitID(), data["commitId"])
	for _, key := range []string{"hardwareId", "architecture", "cpuModel", "osVersion", "kernelVersion"} {
		assert.Contains(t, data, key)
	}
}

func TestUpdateGatewayMetaHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager, _ := newGatewayTestManager(t)
	engine := gin.New()
	InstallHandler(engine.Group("/api/v1"), manager)

	rec := serveGateway(engine, http.MethodPut, "/api/v1/gatewayMeta", `{"name":"plant gateway"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var result response.Result
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Equal(t, response.SuccessCode, result.Code)
	data := result.Data.(map[string]interface{})
	assert.Equal(t, "plant gateway", data["name"])

	for _, body := range []string{`{"name":""}`, `{"name":"bad/name"}`} {
		rec = serveGateway(engine, http.MethodPut, "/api/v1/gatewayMeta", body)
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
		assert.Equal(t, int(response.ErrCodeGatewayMetaInvalid), result.Code)
		require.NotEmpty(t, result.Errors)
		assert.Equal(t, "name", result.Errors[0].Field)
	}
}

func serveGateway(engine *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}
