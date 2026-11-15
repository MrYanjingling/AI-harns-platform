package logmgr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/apis/response"
)

func TestExportLogsFiltersLevel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	path := filepath.Join(t.TempDir(), "gateway.log")
	content := "" +
		"time=2026-01-01T00:00:00Z level=INFO msg=info line\n" +
		"time=2026-01-01T00:00:01Z level=WARN msg=warn line\n" +
		"time=2026-01-01T00:00:02Z level=ERROR msg=error line\n" +
		`{"time":"2026-01-01T00:00:03Z","level":"ERROR","msg":"json error"}` + "\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	engine := gin.New()
	InstallHandler(engine.Group("/api/v1"), path)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/logs/export?level=error", nil)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/plain")
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "harns-gateway-error.log")
	assert.Contains(t, rec.Body.String(), "msg=error line")
	assert.Contains(t, rec.Body.String(), "json error")
	assert.NotContains(t, rec.Body.String(), "msg=info line")
	assert.NotContains(t, rec.Body.String(), "msg=warn line")
}

func TestExportLogsRejectsInvalidLevel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	InstallHandler(engine.Group("/api/v1"), filepath.Join(t.TempDir(), "gateway.log"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/logs/export?level=debug", nil)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	var result response.Result
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Equal(t, int(response.ErrCodeSystemLogLevelInvalid), result.Code)
}

func TestExportLogsRejectsUnavailableFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	InstallHandler(engine.Group("/api/v1"), filepath.Join(t.TempDir(), "missing.log"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/logs/export?level=info", nil)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	var result response.Result
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Equal(t, int(response.ErrCodeSystemLogFileUnavailable), result.Code)
}
