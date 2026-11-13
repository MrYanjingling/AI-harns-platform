package usermgr

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

	"harns-platform/pkg/apis/response"
	v1 "harns-platform/pkg/gateway/usermgr/v1"
)

func newTestEngine(t *testing.T) (*Service, *Store, *gin.Engine) {
	t.Helper()
	return newTestEngineWithAuth(t, true)
}

func newTestEngineWithAuth(t *testing.T, authEnabled bool) (*Service, *Store, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	service, store := newBootstrappedService(t)
	service = NewService(store, authEnabled)
	engine := gin.New()
	v1Group := engine.Group("/api/v1")
	InstallPublicHandler(v1Group, service)
	protected := v1Group.Group("")
	protected.Use(service.RequireAuth())
	InstallHandler(protected, service)
	return service, store, engine
}

func TestServerRequiresTokenAndLoginReturnsUser(t *testing.T) {
	_, _, engine := newTestEngine(t)

	rec := doJSON(engine, http.MethodGet, "/api/v1/users", "", "")
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assertResponseCode(t, rec, response.ErrCodeLoginExpired)

	rec = doJSON(engine, http.MethodPost, "/api/v1/auth/login", "", `{"username":"admin","password":"Harns@123"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var login response.Result
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &login))
	assert.Equal(t, response.SuccessCode, login.Code)
	data, ok := login.Data.(map[string]interface{})
	require.True(t, ok)
	token, _ := data["token"].(string)
	assert.Len(t, token, 64)
	user, ok := data["user"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "admin", user["username"])
	assert.Equal(t, true, user["isAdmin"])
	assert.NotContains(t, rec.Body.String(), "password")

	rec = doJSON(engine, http.MethodGet, "/api/v1/users", token, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func TestServerAdminPermissionsAndUserProtection(t *testing.T) {
	_, store, engine := newTestEngine(t)
	admin, err := store.GetByUsername(context.Background(), SuperuserName)
	require.NoError(t, err)
	adminToken := loginToken(t, engine, SuperuserName, SuperuserPassword)

	rec := doJSON(engine, http.MethodPost, "/api/v1/users", adminToken, `{"username":"operator","password":"secret1","remark":"one"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var created response.Result
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	createdData := created.Data.(map[string]interface{})
	userID := createdData["id"].(string)
	assert.Equal(t, "operator", createdData["username"])
	assert.NotContains(t, rec.Body.String(), "password")

	operatorToken := loginToken(t, engine, "operator", "secret1")
	for _, tc := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPut, "/api/v1/users/" + admin.ID, `{"remark":"changed"}`},
		{http.MethodDelete, "/api/v1/users/" + admin.ID, ""},
		{http.MethodPost, "/api/v1/users/" + admin.ID + "/password/reset", ""},
	} {
		rec = doJSON(engine, tc.method, tc.path, operatorToken, tc.body)
		assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
		assertResponseCode(t, rec, response.ErrCodeForbidden)
	}

	rec = doJSON(engine, http.MethodPut, "/api/v1/users/"+userID, adminToken, `{"remark":"two"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	rec = doJSON(engine, http.MethodPost, "/api/v1/users/"+userID+"/password/reset", adminToken, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.NotContains(t, rec.Body.String(), "password")

	rec = doJSON(engine, http.MethodDelete, "/api/v1/users/"+admin.ID, adminToken, "")
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assertResponseCode(t, rec, response.ErrCodeUserProtected)

	rec = doJSON(engine, http.MethodDelete, "/api/v1/users/"+userID, adminToken, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func TestServerChangePasswordIsSelfOnly(t *testing.T) {
	_, _, engine := newTestEngine(t)
	adminToken := loginToken(t, engine, SuperuserName, SuperuserPassword)
	rec := doJSON(engine, http.MethodPost, "/api/v1/users", adminToken, `{"username":"operator","password":"secret1"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var created response.Result
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	userID := created.Data.(map[string]interface{})["id"].(string)

	operatorToken := loginToken(t, engine, "operator", "secret1")
	rec = doJSON(engine, http.MethodPut, "/api/v1/users/"+userID+"/password", operatorToken,
		`{"oldPassword":"wrong","newPassword":"NewPass1"}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assertResponseCode(t, rec, response.ErrCodeUserPasswordMismatch)

	rec = doJSON(engine, http.MethodPut, "/api/v1/users/"+userID+"/password", operatorToken,
		`{"oldPassword":"secret1","newPassword":"NewPass1"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.NotContains(t, rec.Body.String(), "password")

	rec = doJSON(engine, http.MethodGet, "/api/v1/users", operatorToken, "")
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assertResponseCode(t, rec, response.ErrCodeLoginExpired)

	loginToken(t, engine, "operator", "NewPass1")
}

func loginToken(t *testing.T, engine *gin.Engine, username, password string) string {
	t.Helper()
	body, err := json.Marshal(v1.LoginReq{Username: username, Password: password})
	require.NoError(t, err)
	rec := doJSON(engine, http.MethodPost, "/api/v1/auth/login", "", string(body))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var result response.Result
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)
	token, _ := data["token"].(string)
	require.NotEmpty(t, token)
	return token
}

func doJSON(engine *gin.Engine, method, path, token, body string) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func assertResponseCode(t *testing.T, rec *httptest.ResponseRecorder, code response.ErrCode) {
	t.Helper()
	var result response.Result
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Equal(t, int(code), result.Code, rec.Body.String())
}

func TestServerAuthCanBeDisabled(t *testing.T) {
	_, _, engine := newTestEngineWithAuth(t, false)

	rec := doJSON(engine, http.MethodGet, "/api/v1/users", "", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = doJSON(engine, http.MethodPost, "/api/v1/users", "", `{"username":"operator","password":"secret1"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var created response.Result
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	userID := created.Data.(map[string]interface{})["id"].(string)

	rec = doJSON(engine, http.MethodPut, "/api/v1/users/"+userID, "", `{"remark":"debug"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	rec = doJSON(engine, http.MethodPost, "/api/v1/users/"+userID+"/password/reset", "", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	rec = doJSON(engine, http.MethodDelete, "/api/v1/users/"+userID, "", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}
