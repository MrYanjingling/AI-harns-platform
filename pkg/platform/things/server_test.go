package things

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallHandlerRegistersRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	InstallHandler(engine.Group("/api/v1/iot"), &Service{})

	found := make(map[string]bool)
	for _, route := range engine.Routes() {
		found[route.Method+" "+route.Path] = true
	}
	for _, want := range []string{
		"POST /api/v1/iot/things",
		"POST /api/v1/iot/things/timeseries",
		"GET /api/v1/iot/things",
		"GET /api/v1/iot/things/:id/basic",
		"GET /api/v1/iot/things/:id/features",
		"GET /api/v1/iot/things/:id/shadow",
		"PUT /api/v1/iot/things/:id",
		"DELETE /api/v1/iot/things",
	} {
		assert.True(t, found[want], "missing route %s", want)
	}
}

func TestWriteErrReturnsFieldErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	writeErr(c, NewValidationError([]FieldIssue{{
		Field: "things[0].deviceName",
		MsgID: "validate.thing_device_name_required",
	}}))

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"field":"things[0].deviceName"`)
	assert.Contains(t, recorder.Body.String(), `"code":10054`)
}

func TestNewServiceDefaultTTL(t *testing.T) {
	svc := NewService(nil, nil, 0, nil, nil, nil, AuthDenyCacheConfig{})
	require.NotNil(t, svc)
	assert.Equal(t, 60, int(svc.onlineTTL.Seconds()))
}

func TestDeviceAuthSharedSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	InstallEMQXHandler(
		engine.Group("/api/v1/iot"),
		&Service{},
		NewMessagePipeline(MessagePipelineConfig{}, NewDefaultMessageRouter()),
		"0123456789abcdef",
	)

	do := func(secret, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/iot/thingsAuth", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if secret != "" {
			req.Header.Set(AuthSecretHeader, secret)
		}
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		return rec
	}

	missing := do("", `{}`)
	assert.Equal(t, http.StatusUnauthorized, missing.Code)
	assert.Contains(t, missing.Body.String(), `"result":"deny"`)

	wrong := do("wrong-secret", `{}`)
	assert.Equal(t, http.StatusUnauthorized, wrong.Code)

	emptyCreds := do("0123456789abcdef", `{}`)
	assert.Equal(t, http.StatusUnauthorized, emptyCreds.Code)
	assert.Contains(t, emptyCreds.Body.String(), `"result":"deny"`)
}

func TestBuildAuthResponse(t *testing.T) {
	item := &AuthThing{
		Thing: Thing{
			DeviceKey:  "device_001",
			DeviceName: "设备1",
			UpTopic:    "iot/device_001/up",
			DownTopic:  "iot/device_001/down",
		},
		ThingModelKey: "tm_test",
	}
	item.ID = "thing-1"
	item.ThingModelID = "model-1"

	resp := buildAuthResponse(item)
	require.Equal(t, "allow", resp.Result)
	assert.False(t, resp.Superuser)
	assert.Equal(t, "tm_test", resp.ClientAttrs["thing_model_key"])
	assert.Equal(t, "device_001", resp.ClientAttrs["device_key"])
	require.Len(t, resp.ACL, 2)
	assert.Equal(t, "iot/device_001/up", resp.ACL[0].Topic)
	assert.Equal(t, "iot/device_001/down", resp.ACL[1].Topic)
	for _, rule := range resp.ACL {
		assert.Equal(t, "allow", rule.Permission)
		assert.Equal(t, "all", rule.Action)
		assert.Equal(t, []int{0, 1, 2}, rule.QoS)
		assert.False(t, rule.Retain)
	}
}
