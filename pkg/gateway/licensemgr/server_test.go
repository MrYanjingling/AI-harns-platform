package licensemgr

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/apis/response"
)

func newTestEngine(t *testing.T) (*Service, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	svc, _, _ := newTestService(t, publicKeyFile(t))
	engine := gin.New()
	admin := func(c *gin.Context) {
		if c.GetHeader("X-Admin") != "true" {
			response.WriteError(c, http.StatusBadRequest, response.ErrForbidden())
			c.Abort()
			return
		}
		c.Next()
	}
	InstallHandler(engine.Group("/api/v1"), svc, admin)
	return svc, engine
}

func licenseRequest(t *testing.T, engine *gin.Engine, method, path, data string, admin bool) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(ImportRequest{Data: data})
	require.NoError(t, err)
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if admin {
		req.Header.Set("X-Admin", "true")
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func TestServerQueryUnauthorizedAndImport(t *testing.T) {
	_, engine := newTestEngine(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/license", nil)
	engine.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var result response.Result
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Equal(t, StatusUnauthorized, result.Data.(map[string]interface{})["status"])

	data := signedLicense(t, validClaims(100))
	rec = licenseRequest(t, engine, http.MethodPost, "/api/v1/license/import", data, true)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	imported := result.Data.(map[string]interface{})
	assert.Equal(t, StatusAuthorized, imported["status"])
	license := imported["license"].(map[string]interface{})
	assert.Equal(t, float64(100), license["maxVariable"])
}

func TestServerImportRequiresAdmin(t *testing.T) {
	_, engine := newTestEngine(t)
	data := signedLicense(t, validClaims(100))
	rec := licenseRequest(t, engine, http.MethodPost, "/api/v1/license/import", data, false)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	var result response.Result
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Equal(t, int(response.ErrCodeForbidden), result.Code)
}

func TestServerImportRejectsEmptyData(t *testing.T) {
	_, engine := newTestEngine(t)
	rec := licenseRequest(t, engine, http.MethodPost, "/api/v1/license/import", "  ", true)
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	var result response.Result
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Equal(t, int(response.ErrCodeLicenseInvalid), result.Code)
	require.NotEmpty(t, result.Errors)
	assert.Equal(t, "data", result.Errors[0].Field)
}

func TestServerImportRejectsInvalidSignature(t *testing.T) {
	_, engine := newTestEngine(t)
	data := signedLicense(t, validClaims(100))
	rec := licenseRequest(t, engine, http.MethodPost, "/api/v1/license/import", data+"x", true)
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	var result response.Result
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Equal(t, int(response.ErrCodeLicenseInvalid), result.Code)
}
