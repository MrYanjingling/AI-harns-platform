package emqx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

func TestBroker(t *testing.T) {
	svc := NewService(Config{Broker: "broker:1883"})
	assert.Equal(t, "broker:1883", svc.Broker())
	var nilService *Service
	assert.Empty(t, nilService.Broker())
}

func TestKickClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v5/clients/client-1", r.URL.Path)
		assert.Equal(t, http.MethodDelete, r.Method)
		username, password, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "ak", username)
		assert.Equal(t, "sk", password)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	svc := NewService(Config{APIURL: server.URL, AK: "ak", SK: "sk"})
	assert.NoError(t, svc.KickClient(context.Background(), "client-1"))
}

func TestKickClientNotFoundIsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc := NewService(Config{APIURL: server.URL, AK: "ak", SK: "sk"})
	assert.NoError(t, svc.KickClient(context.Background(), "missing"))
}

func TestKickClientNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	svc := NewService(Config{APIURL: server.URL, AK: "ak", SK: "sk"})
	require.Error(t, svc.KickClient(context.Background(), "client-1"))
}

func TestKickClientsUsesBulkEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v5/clients/kickout/bulk", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		username, password, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "ak", username)
		assert.Equal(t, "sk", password)

		var ids []string
		require.NoError(t, json.NewDecoder(r.Body).Decode(&ids))
		assert.Equal(t, []string{"a", "b"}, ids)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	svc := NewService(Config{APIURL: server.URL, AK: "ak", SK: "sk"})
	assert.NoError(t, svc.KickClients(context.Background(), []string{"a", "a", " ", "b"}))
}

func TestKickClientsFallsBackToIndividualDelete(t *testing.T) {
	var mu sync.Mutex
	seen := make(map[string]int)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v5/clients/kickout/bulk" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		clientID := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		mu.Lock()
		seen[clientID]++
		mu.Unlock()
		if clientID == "bad" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	svc := NewService(Config{APIURL: server.URL, AK: "ak", SK: "sk"})
	err := svc.KickClients(context.Background(), []string{"a", "a", " ", "bad", "b"})
	require.Error(t, err)
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, seen["a"])
	assert.Equal(t, 1, seen["bad"])
	assert.Equal(t, 1, seen["b"])
}
