package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/cmd/gateway/config"
	"harns-platform/cmd/gateway/options"
	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/sqlite"
	"harns-platform/pkg/gateway/usermgr"
	"harns-platform/pkg/generic"
)

func TestGatewayAPIAuthenticationWiring(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	store := usermgr.NewStore(db)
	require.NoError(t, store.Initialize(context.Background()))
	users := usermgr.NewService(store, true)
	require.NoError(t, users.BootstrapSuperuser(context.Background()))

	server, err := NewServer(gin.New(), options.NewDefaultOptions(), &config.Config{
		UserMgr:      users,
		MaxBodyBytes: 1 << 20,
		CORS:         generic.DefaultCORSConfig(),
	})
	require.NoError(t, err)

	rec := serve(server.Router, http.MethodGet, "/api/v1/gatewayMeta", "", "")
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assertCode(t, rec, int(response.ErrCodeLoginExpired))

	rec = serve(server.Router, http.MethodPost, "/api/v1/auth/login", "", `{"username":"admin","password":"Harns@123"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var result response.Result
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	token := result.Data.(map[string]interface{})["token"].(string)
	require.NotEmpty(t, token)

	rec = serve(server.Router, http.MethodGet, "/api/v1/users", token, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func serve(router *gin.Engine, method, path, token, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func assertCode(t *testing.T, rec *httptest.ResponseRecorder, code int) {
	t.Helper()
	var result response.Result
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Equal(t, code, result.Code, rec.Body.String())
}
