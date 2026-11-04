package timeseries

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureDatabase(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{"created", http.StatusOK, `{}`, false},
		{"exists-conflict", http.StatusConflict, `{"message":"database already exists"}`, false},
		{"exists-body", http.StatusBadRequest, `database already exists`, false},
		{"failed", http.StatusInternalServerError, `boom`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v3/configure/database", r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			cfg := DefaultConfig()
			cfg.URL = server.URL
			cfg.Token = "test-token"
			store, err := NewStore(cfg)
			require.NoError(t, err)
			t.Cleanup(func() { _ = store.Close() })

			err = store.ensureDatabase(context.Background(), cfg.Database, cfg.DataRetention)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuildLineProtocol(t *testing.T) {
	points := []*Point{
		{
			Measurement: "device_property_tm_1",
			Tags:        map[string]string{"model_id": "m1", "device_id": "d1", "device_key": "k 1"},
			Fields:      map[string]any{"a": int64(1), "b": "x", "c": true, "d": 1.5, "skip": nil},
			Time:        time.UnixMilli(1000),
		},
	}
	got := string(buildLineProtocol(points))
	assert.Contains(t, got, "device_property_tm_1,device_id=d1,device_key=k\\ 1,model_id=m1 ")
	assert.Contains(t, got, "a=1i")
	assert.Contains(t, got, `b="x"`)
	assert.Contains(t, got, "c=true")
	assert.Contains(t, got, "d=1.5")
	assert.NotContains(t, got, "skip=")
	assert.True(t, strings.HasSuffix(got, " 1000\n"), got)
}

func TestBuildLineProtocolSkipsPointWithoutFields(t *testing.T) {
	points := []*Point{{Measurement: "m", Tags: map[string]string{"a": "b"}, Fields: map[string]any{"x": nil}, Time: time.Now()}}
	assert.Empty(t, buildLineProtocol(points))
}

func TestWritePointsHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v3/write_lp", r.URL.Path)
		assert.Equal(t, "harns_timeseries", r.URL.Query().Get("db"))
		assert.Equal(t, "millisecond", r.URL.Query().Get("precision"))
		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), "device_property_tm_1")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.URL = server.URL
	cfg.Token = "test-token"
	store, err := NewStore(cfg)
	require.NoError(t, err)

	err = store.writePoints(context.Background(), cfg.Database, []*Point{{
		Measurement: "device_property_tm_1",
		Tags:        map[string]string{"device_id": "d1"},
		Fields:      map[string]any{"a": int64(1)},
		Time:        time.UnixMilli(1000),
	}})
	assert.NoError(t, err)
}
