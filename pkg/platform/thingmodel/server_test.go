package thingmodel

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/platform/thingmodel/tsl"
)

func TestInstallHandlerRegistersRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	InstallHandler(engine.Group("/api/v1/iot"), &Service{})

	routes := engine.Routes()
	require.NotEmpty(t, routes)

	found := make(map[string]bool, len(routes))
	for _, route := range routes {
		found[route.Method+" "+route.Path] = true
	}
	for _, want := range []string{
		"POST /api/v1/iot/thingModels",
		"GET /api/v1/iot/thingModels",
		"GET /api/v1/iot/thingModels/:id",
		"POST /api/v1/iot/thingModels/:id/properties",
		"DELETE /api/v1/iot/thingModels/:id/properties",
		"GET /api/v1/iot/thingModels/:id/tsl",
	} {
		assert.True(t, found[want], "missing route %s", want)
	}
}

func TestWriteErrReturnsFieldErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	writeErr(c, tsl.NewValidationError([]tsl.FieldIssue{{
		Field: "properties[0].identifier",
		MsgID: "validate.thing_model_identifier_required",
	}}))

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"field":"properties[0].identifier"`)
	assert.Contains(t, recorder.Body.String(), `"code":10045`)
}
