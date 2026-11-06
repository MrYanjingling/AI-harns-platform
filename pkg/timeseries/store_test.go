package timeseries

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

func TestQuerySQL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v3/query_sql", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		var body map[string]string
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "harns_timeseries", body["db"])
		assert.Equal(t, "SELECT 1", body["q"])
		assert.Equal(t, "json", body["format"])
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"device_id":"d1","value":26.5}]`))
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.URL = server.URL
	cfg.Token = "test-token"
	store, err := NewStore(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	rows, err := store.QuerySQL(context.Background(), cfg.Database, "SELECT 1")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "d1", rows[0]["device_id"])
	assert.Equal(t, json.Number("26.5"), rows[0]["value"])
}

func TestDeleteTable(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{"deleted", http.StatusNoContent, "", false},
		{"not-found", http.StatusNotFound, `{"message":"table not found"}`, false},
		{"failed", http.StatusInternalServerError, "boom", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v3/configure/table", r.URL.Path)
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, "harns_timeseries", r.URL.Query().Get("db"))
				assert.Equal(t, "device_property_tm_demo", r.URL.Query().Get("table"))
				assert.Empty(t, r.URL.Query().Get("hard_delete_at"))
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			cfg := DefaultConfig()
			cfg.URL = server.URL
			store, err := NewStore(cfg)
			require.NoError(t, err)
			t.Cleanup(func() { _ = store.Close() })

			err = store.DeleteTable(context.Background(), cfg.Database, "device_property_tm_demo")
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDropThingModelTables(t *testing.T) {
	var mu sync.Mutex
	deleted := make(map[string]bool)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v3/configure/database" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/api/v3/configure/table" && r.Method == http.MethodDelete:
			database := r.URL.Query().Get("db")
			table := r.URL.Query().Get("table")
			mu.Lock()
			deleted[database+"."+table] = true
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.URL = server.URL
	store, err := NewStore(cfg)
	require.NoError(t, err)
	require.NoError(t, store.Start(context.Background()))
	t.Cleanup(func() { _ = store.Close() })

	svc := &Service{cfg: cfg, store: store}
	require.NoError(t, svc.DropThingModelTables(context.Background(), "tm_demo"))

	mu.Lock()
	defer mu.Unlock()
	want := []string{
		"harns_timeseries.device_property_tm_demo",
		"harns_timeseries.device_event_tm_demo",
		"harns_timeseries.device_command_record_tm_demo",
		"harns_timeseries_log.device_property_log_tm_demo",
		"harns_timeseries_log.device_event_log_tm_demo",
		"harns_timeseries_debug.device_property_tm_demo",
	}
	for _, table := range want {
		assert.True(t, deleted[table], "missing delete for %s", table)
	}
	assert.Len(t, deleted, len(want))
}
