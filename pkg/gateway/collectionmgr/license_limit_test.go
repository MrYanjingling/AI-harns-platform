package collectionmgr

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/validation"
)

type fixedVariableLimit int

func (f fixedVariableLimit) MaxVariable() int { return int(f) }

type fixedLicenseStatus bool

func (f fixedLicenseStatus) IsAuthorized() bool { return bool(f) }

func TestCreateCNCCollectionRequiresAuthorizedLicense(t *testing.T) {
	service, _ := newTestService(t)
	service.SetLicenseStatusProvider(fixedLicenseStatus(false))

	_, err := service.Create(context.Background(), []byte(`{"driverType":"CNC","driverModel":"KND"}`))
	require.Error(t, err)
	var validationErr *validation.ValidationError
	require.True(t, errors.As(err, &validationErr))
	require.Len(t, validationErr.Issues, 1)
	assert.Equal(t, "driverType", validationErr.Issues[0].Field)
	assert.Equal(t, "validate.collection_cnc_license_required", validationErr.Issues[0].MsgID)
}

func TestUnauthorizedCollectionLimit(t *testing.T) {
	service, _ := newTestService(t)
	for i := 0; i < unauthorizedCollectionLimit; i++ {
		_, err := service.Create(context.Background(), []byte(collectionBody))
		require.NoError(t, err)
	}

	_, err := service.Create(context.Background(), []byte(collectionBody))
	require.Error(t, err)
	var validationErr *validation.ValidationError
	require.True(t, errors.As(err, &validationErr))
	require.Len(t, validationErr.Issues, 1)
	assert.Equal(t, "collection", validationErr.Issues[0].Field)
	assert.Equal(t, "validate.collection_unauthorized_limit", validationErr.Issues[0].MsgID)
}

func TestCreateCollectionEnforcesLicenseVariableLimit(t *testing.T) {
	service, _ := newTestService(t)
	service.SetVariableLimitProvider(fixedVariableLimit(1))
	engine := gin.New()
	InstallHandler(engine.Group("/api/v1"), service)

	body := `{
		"driverType":"modbus","driverModel":"modbusTcp","name":"limited",
		"collectorCycleMs":1000,"protocolConfig":{"location":"127.0.0.1:502","memoryLayout":"ABCD"},
		"variables":[
			{"name":"temperature","dataType":"float32","address":"1!40001","accessMode":"r"},
			{"name":"humidity","dataType":"float32","address":"1!40003","accessMode":"r"}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"field":"variables"`)
}

func TestAddVariablesEnforcesLicenseVariableLimit(t *testing.T) {
	service, _ := newTestService(t)
	service.SetVariableLimitProvider(fixedVariableLimit(1))
	engine := gin.New()
	InstallHandler(engine.Group("/api/v1"), service)
	id := createTestCollection(t, engine, collectionBody)

	addBody := `[{"name":"humidity","dataType":"float32","address":"1!40002","accessMode":"r"}]`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections/"+id+"/variables", strings.NewReader(addBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"field":"variables"`)
}

func TestAddCalculatedVariableDoesNotCountTowardLicenseLimit(t *testing.T) {
	service, _ := newTestService(t)
	service.SetVariableLimitProvider(fixedVariableLimit(1))
	engine := gin.New()
	InstallHandler(engine.Group("/api/v1"), service)
	id := createTestCollection(t, engine, collectionBody)

	addBody := `[{"name":"energy","dataType":"float32","address":"","accessMode":"r","isCalculated":true,"expression":"temperature * 2"}]`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections/"+id+"/variables", strings.NewReader(addBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"added":1`)
}
