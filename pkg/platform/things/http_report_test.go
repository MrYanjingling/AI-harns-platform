package things

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/apis/response"
)

func TestHTTPReportMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	sink := &stubSink{}
	InstallHTTPReportHandler(engine.Group("/api/v1/iot"), &Service{}, sink)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/iot/thingsData", strings.NewReader(`{"a":1}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":10031`)
	assert.False(t, sink.called)
}

func TestHTTPReportMalformedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	sink := &stubSink{}
	InstallHTTPReportHandler(engine.Group("/api/v1/iot"), &Service{}, sink)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/iot/thingsData", strings.NewReader("not-json"))
	req.Header.Set(TokenHeader, "token")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.False(t, sink.called)
}

func TestTokenCacheKeyHidesToken(t *testing.T) {
	key := tokenCacheKey("secret-token")
	assert.True(t, strings.HasPrefix(key, tokenCachePrefix))
	assert.NotContains(t, key, "secret-token")
	assert.Equal(t, key, tokenCacheKey("secret-token"))
}

func TestGetThingByTokenEmpty(t *testing.T) {
	svc := NewService(nil, nil, 0, nil, nil, nil, AuthDenyCacheConfig{})
	_, err := svc.GetThingByToken(context.Background(), " ")
	require.Error(t, err)
	assert.Equal(t, int(response.ErrCodeUnauthorized), response.ErrorCode(err))
}

func TestTouchThingOnlineWithoutRedis(t *testing.T) {
	svc := NewService(nil, nil, 0, nil, nil, nil, AuthDenyCacheConfig{})
	svc.TouchThingOnline(context.Background(), "thing-1", time.Now().UnixMilli())
}

func TestHTTPReportInvalidType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	sink := &stubSink{}
	InstallHTTPReportHandler(engine.Group("/api/v1/iot"), &Service{}, sink)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/iot/thingsData", strings.NewReader(`{"type":"command"}`))
	req.Header.Set(TokenHeader, "token")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.False(t, sink.called)
}

func TestHTTPReportMissingType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	sink := &stubSink{}
	InstallHTTPReportHandler(engine.Group("/api/v1/iot"), &Service{}, sink)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/iot/thingsData", strings.NewReader(`{"a":1}`))
	req.Header.Set(TokenHeader, "token")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.False(t, sink.called)
}
