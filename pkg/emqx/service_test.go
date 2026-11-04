package emqx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublish(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v5/publish", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		username, password, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "ak", username)
		assert.Equal(t, "sk", password)

		var body PublishRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "sensor/temp", body.Topic)
		assert.Equal(t, "25.5", body.Payload)
		assert.Equal(t, 1, body.QoS)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	svc := NewService(Config{APIURL: server.URL, AK: "ak", SK: "sk"})
	err := svc.Publish(context.Background(), PublishRequest{
		Topic:   "sensor/temp",
		Payload: "25.5",
		QoS:     1,
	})
	assert.NoError(t, err)
}

func TestPublishValidation(t *testing.T) {
	svc := NewService(DefaultConfig())

	assert.Error(t, svc.Publish(context.Background(), PublishRequest{Topic: "", Payload: "x", QoS: 0}))
	assert.Error(t, svc.Publish(context.Background(), PublishRequest{Topic: "t", Payload: "x", QoS: 3}))
}

func TestPublishNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"bad credentials"}`))
	}))
	defer server.Close()

	svc := NewService(Config{APIURL: server.URL, AK: "ak", SK: "sk"})
	err := svc.Publish(context.Background(), PublishRequest{Topic: "t", Payload: "x", QoS: 0})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "http://127.0.0.1:18083", cfg.APIURL)
	assert.Equal(t, 10000, cfg.TimeoutMs)
}
