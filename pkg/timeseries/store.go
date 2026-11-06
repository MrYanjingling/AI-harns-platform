package timeseries

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"harns-platform/pkg/logging"
)

// Store wraps the InfluxDB HTTP writer and the batch pipelines.
type Store struct {
	cfg   Config
	http  *http.Client
	data  *batcher
	debug *batcher
	log   *batcher
}

// NewStore creates the HTTP writer and batchers.
func NewStore(cfg Config) (*Store, error) {
	s := &Store{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.WriteTimeout()},
	}
	s.data = newBatcher(&databaseWriter{store: s, database: cfg.Database},
		cfg.BatchSize, cfg.FlushInterval(), cfg.WriteTimeout(), cfg.QueueSize)
	if cfg.DebugDatabase != "" {
		s.debug = newBatcher(&databaseWriter{store: s, database: cfg.DebugDatabase},
			cfg.BatchSize, cfg.FlushInterval(), cfg.WriteTimeout(), cfg.QueueSize)
	}
	s.log = newBatcher(&databaseWriter{store: s, database: cfg.LogDatabase},
		cfg.BatchSize, cfg.FlushInterval(), cfg.WriteTimeout(), cfg.QueueSize)
	return s, nil
}

// Start ensures the databases exist and starts the batch workers.
func (s *Store) Start(ctx context.Context) error {
	if err := s.ensureDatabase(ctx, s.cfg.Database, s.cfg.DataRetention); err != nil {
		return err
	}
	if s.cfg.DebugDatabase != "" {
		if err := s.ensureDatabase(ctx, s.cfg.DebugDatabase, s.cfg.DebugDataRetention); err != nil {
			return err
		}
	}
	if err := s.ensureDatabase(ctx, s.cfg.LogDatabase, s.cfg.LogRetention); err != nil {
		return err
	}
	s.data.start()
	if s.debug != nil {
		s.debug.start()
	}
	s.log.start()
	return nil
}

// Close drains the batch workers.
func (s *Store) Close() error {
	if s.data != nil {
		s.data.close()
	}
	if s.debug != nil {
		s.debug.close()
	}
	if s.log != nil {
		s.log.close()
	}
	return nil
}

// EnqueueData queues points for the data database.
func (s *Store) EnqueueData(ctx context.Context, points ...*Point) error {
	if s.data == nil {
		return ErrBatcherClosed
	}
	return s.data.enqueue(ctx, points...)
}

// EnqueueDebugData queues points for the debug database.
func (s *Store) EnqueueDebugData(ctx context.Context, points ...*Point) error {
	if s.debug == nil {
		return ErrBatcherClosed
	}
	return s.debug.enqueue(ctx, points...)
}

// EnqueueLog queues points for the log database.
func (s *Store) EnqueueLog(ctx context.Context, points ...*Point) error {
	if s.log == nil {
		return ErrBatcherClosed
	}
	return s.log.enqueue(ctx, points...)
}

// Flush waits until the current data, debug, and log batches are written.
func (s *Store) Flush(ctx context.Context) error {
	for _, b := range []*batcher{s.data, s.debug, s.log} {
		if b == nil {
			continue
		}
		if err := b.flushNow(ctx); err != nil {
			return err
		}
	}
	return nil
}

type databaseWriter struct {
	store    *Store
	database string
}

func (w *databaseWriter) WritePoints(ctx context.Context, points []*Point) error {
	return w.store.writePoints(ctx, w.database, points)
}

func (s *Store) writePoints(ctx context.Context, database string, points []*Point) error {
	body := buildLineProtocol(points)
	if len(body) == 0 {
		return nil
	}

	endpoint := strings.TrimRight(s.cfg.URL, "/") + "/api/v3/write_lp?db=" +
		url.QueryEscape(database) + "&precision=millisecond"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	if s.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.Token)
	}

	resp, err := s.http.Do(req)
	if err != nil {
		return fmt.Errorf("write timeseries %s: %w", database, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("write timeseries %s failed: status=%d body=%s",
		database, resp.StatusCode, strings.TrimSpace(string(payload)))
}

// QuerySQL executes a SQL query and returns rows with JSON numbers preserved.
func (s *Store) QuerySQL(ctx context.Context, database, query string) ([]map[string]any, error) {
	body, err := json.Marshal(map[string]string{
		"db":     database,
		"q":      query,
		"format": "json",
	})
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimRight(s.cfg.URL, "/") + "/api/v3/query_sql"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.Token)
	}

	resp, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query timeseries: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("query timeseries failed: status=%d body=%s",
			resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	var rows []map[string]any
	if err := decoder.Decode(&rows); err != nil {
		if err == io.EOF {
			return []map[string]any{}, nil
		}
		return nil, fmt.Errorf("decode timeseries query result: %w", err)
	}
	return rows, nil
}

// QueryColumnNames returns the existing columns of a table.
func (s *Store) QueryColumnNames(ctx context.Context, database, table string) (map[string]struct{}, error) {
	query := fmt.Sprintf(
		`SELECT column_name FROM information_schema.columns WHERE table_schema = 'iox' AND table_name = %s`,
		quoteLiteral(table))
	rows, err := s.QuerySQL(ctx, database, query)
	if err != nil {
		return nil, err
	}
	columns := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		name, _ := row["column_name"].(string)
		if name != "" {
			columns[name] = struct{}{}
		}
	}
	return columns, nil
}

// DeleteTable soft-deletes one measurement from a database.
func (s *Store) DeleteTable(ctx context.Context, database, table string) error {
	query := url.Values{
		"db":    []string{database},
		"table": []string{table},
	}
	endpoint := strings.TrimRight(s.cfg.URL, "/") + "/api/v3/configure/table?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}
	if s.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.Token)
	}

	resp, err := s.http.Do(req)
	if err != nil {
		return fmt.Errorf("delete timeseries table %s.%s: %w", database, table, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	message := strings.ToLower(string(payload))
	if resp.StatusCode == http.StatusNotFound || strings.Contains(message, "not found") || strings.Contains(message, "does not exist") {
		return nil
	}
	return fmt.Errorf("delete timeseries table %s.%s failed: status=%d body=%s",
		database, table, resp.StatusCode, strings.TrimSpace(string(payload)))
}

func (s *Store) ensureDatabase(ctx context.Context, name, retention string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("timeseries: database name is empty")
	}
	body, err := json.Marshal(map[string]string{
		"db":               name,
		"retention_period": retention,
	})
	if err != nil {
		return err
	}

	endpoint := strings.TrimRight(s.cfg.URL, "/") + "/api/v3/configure/database"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.Token)
	}

	resp, err := s.http.Do(req)
	if err != nil {
		return fmt.Errorf("create timeseries database %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logging.V(1).InfoS("Timeseries database ensured", "database", name, "retention", retention)
		return nil
	}

	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	message := strings.ToLower(string(payload))
	if resp.StatusCode == http.StatusConflict || strings.Contains(message, "already exists") || strings.Contains(message, "exists") {
		logging.Warnf("Timeseries database %s already exists; retention is not verified (%s)", name, retention)
		return nil
	}
	return fmt.Errorf("create timeseries database %s failed: status=%d body=%s", name, resp.StatusCode, strings.TrimSpace(string(payload)))
}

func buildLineProtocol(points []*Point) []byte {
	var b strings.Builder
	for _, point := range points {
		if point == nil || point.Measurement == "" {
			continue
		}
		var line strings.Builder
		line.WriteString(escapeMeasurement(point.Measurement))

		tagKeys := sortedKeys(point.Tags)
		for _, key := range tagKeys {
			line.WriteByte(',')
			line.WriteString(escapeTag(key))
			line.WriteByte('=')
			line.WriteString(escapeTag(point.Tags[key]))
		}

		fieldKeys := sortedFieldKeys(point.Fields)
		first := true
		for _, key := range fieldKeys {
			value := formatFieldValue(point.Fields[key])
			if value == "" {
				continue
			}
			if first {
				line.WriteByte(' ')
				first = false
			} else {
				line.WriteByte(',')
			}
			line.WriteString(escapeTag(key))
			line.WriteByte('=')
			line.WriteString(value)
		}
		if first {
			continue
		}

		line.WriteByte(' ')
		line.WriteString(strconv.FormatInt(point.Time.UnixMilli(), 10))
		b.WriteString(line.String())
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedFieldKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func escapeMeasurement(value string) string {
	return strings.NewReplacer(",", `\,`, " ", `\ `).Replace(value)
}

func escapeTag(value string) string {
	return strings.NewReplacer(",", `\,`, "=", `\=`, " ", `\ `).Replace(value)
}

func formatFieldValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int:
		return strconv.FormatInt(int64(v), 10) + "i"
	case int8:
		return strconv.FormatInt(int64(v), 10) + "i"
	case int16:
		return strconv.FormatInt(int64(v), 10) + "i"
	case int32:
		return strconv.FormatInt(int64(v), 10) + "i"
	case int64:
		return strconv.FormatInt(v, 10) + "i"
	case uint:
		return strconv.FormatUint(uint64(v), 10) + "u"
	case uint8:
		return strconv.FormatUint(uint64(v), 10) + "u"
	case uint16:
		return strconv.FormatUint(uint64(v), 10) + "u"
	case uint32:
		return strconv.FormatUint(uint64(v), 10) + "u"
	case uint64:
		return strconv.FormatUint(v, 10) + "u"
	case float32:
		return formatFloat(float64(v))
	case float64:
		return formatFloat(v)
	case string:
		return quoteString(v)
	default:
		return quoteString(fmt.Sprint(v))
	}
}

func formatFloat(value float64) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return ""
	}
	return strconv.FormatFloat(value, 'g', -1, 64)
}

func quoteString(value string) string {
	var b strings.Builder
	b.Grow(len(value) + 2)
	b.WriteByte('"')
	for _, r := range value {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n', '\r', '\t':
			b.WriteByte(' ')
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
