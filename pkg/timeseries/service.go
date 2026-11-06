package timeseries

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"harns-platform/pkg/logging"
)

// ErrDisabled is returned when the timeseries service is disabled.
var ErrDisabled = errors.New("timeseries: service disabled")

// Service converts business records into InfluxDB points and enqueues them.
type Service struct {
	cfg   Config
	store *Store
}

// NewService creates the timeseries service.
func NewService(cfg Config) (*Service, error) {
	if !cfg.Enabled {
		return &Service{cfg: cfg}, nil
	}
	store, err := NewStore(cfg)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg, store: store}, nil
}

// Start ensures databases and starts the batch workers.
func (s *Service) Start(ctx context.Context) error {
	if !s.cfg.Enabled || s.store == nil {
		return nil
	}
	return s.store.Start(ctx)
}

// Close drains pending points and closes the clients.
func (s *Service) Close() error {
	if s.store == nil {
		return nil
	}
	return s.store.Close()
}

// DropThingModelTables flushes pending writes and deletes all model-scoped
// measurements from the configured data, debug, and log databases.
func (s *Service) DropThingModelTables(ctx context.Context, modelKey string) error {
	if s.store == nil {
		return nil
	}
	if err := s.store.Flush(ctx); err != nil {
		logging.ErrorS(err, "Failed to flush timeseries before table drop", "thingModelKey", modelKey)
	}

	type tableRef struct {
		database string
		table    string
	}
	tables := []tableRef{
		{s.cfg.Database, PropertyTable(modelKey)},
		{s.cfg.Database, EventTable(modelKey)},
		{s.cfg.Database, CommandRecordTable(modelKey)},
		{s.cfg.LogDatabase, PropertyLogTable(modelKey)},
		{s.cfg.LogDatabase, EventLogTable(modelKey)},
	}
	if s.cfg.DebugDatabase != "" {
		tables = append(tables, tableRef{s.cfg.DebugDatabase, PropertyTable(modelKey)})
	}

	var errs []error
	for _, table := range tables {
		if err := s.store.DeleteTable(ctx, table.database, table.table); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("drop thing model tables: %w", errors.Join(errs...))
	}
	return nil
}

// WriteProperty writes property samples.
func (s *Service) WriteProperty(ctx context.Context, modelKey string, points []PropertyPoint, debug bool) error {
	if s.store == nil {
		return ErrDisabled
	}
	table := PropertyTable(modelKey)
	out := make([]*Point, 0, len(points))
	for _, point := range points {
		fields := normalizeFields(point.Values)
		if len(fields) == 0 {
			continue
		}
		out = append(out, &Point{
			Measurement: table,
			Tags: map[string]string{
				TagModelID:   point.ModelID,
				TagDeviceID:  point.DeviceID,
				TagDeviceKey: point.DeviceKey,
			},
			Fields: fields,
			Time:   pointTime(point.Time),
		})
	}
	if debug && s.cfg.DebugDatabase != "" {
		return s.store.EnqueueDebugData(ctx, out...)
	}
	return s.store.EnqueueData(ctx, out...)
}

// WritePropertyLog writes one property request log.
func (s *Service) WritePropertyLog(ctx context.Context, modelKey string, log PropertyLog) error {
	if s.store == nil {
		return ErrDisabled
	}
	point := &Point{
		Measurement: PropertyLogTable(modelKey),
		Tags: map[string]string{
			TagModelID:   log.ModelID,
			TagDeviceID:  log.DeviceID,
			TagDeviceKey: log.DeviceKey,
		},
		Fields: map[string]any{
			FieldSuccess:  log.Success,
			FieldPayload:  log.Payload,
			FieldErrorMsg: log.ErrorMsg,
		},
		Time: pointTime(log.Time),
	}
	return s.store.EnqueueLog(ctx, point)
}

// WriteGatewayPropertyLog writes one gateway property request log.
func (s *Service) WriteGatewayPropertyLog(ctx context.Context, log GatewayPropertyLog) error {
	if s.store == nil {
		return ErrDisabled
	}
	point := &Point{
		Measurement: GatewayPropertyLogMeasurement,
		Tags: map[string]string{
			TagGatewayType: log.GatewayType,
			TagGatewayID:   log.GatewayID,
		},
		Fields: map[string]any{
			FieldSuccess:  log.Success,
			FieldPayload:  log.Payload,
			FieldErrorMsg: log.ErrorMsg,
		},
		Time: pointTime(log.Time),
	}
	return s.store.EnqueueLog(ctx, point)
}

// WriteEvent writes event samples.
func (s *Service) WriteEvent(ctx context.Context, modelKey string, points []EventPoint) error {
	if s.store == nil {
		return ErrDisabled
	}
	table := EventTable(modelKey)
	out := make([]*Point, 0, len(points))
	for _, point := range points {
		out = append(out, &Point{
			Measurement: table,
			Tags: map[string]string{
				TagModelID:       point.ModelID,
				TagDeviceID:      point.DeviceID,
				TagDeviceKey:     point.DeviceKey,
				TagEventIdentify: point.EventIdentify,
			},
			Fields: map[string]any{FieldData: point.Data},
			Time:   pointTime(point.Time),
		})
	}
	return s.store.EnqueueData(ctx, out...)
}

// WriteEventLog writes one event request log.
func (s *Service) WriteEventLog(ctx context.Context, modelKey string, log EventLog) error {
	if s.store == nil {
		return ErrDisabled
	}
	point := &Point{
		Measurement: EventLogTable(modelKey),
		Tags: map[string]string{
			TagModelID:   log.ModelID,
			TagDeviceID:  log.DeviceID,
			TagDeviceKey: log.DeviceKey,
		},
		Fields: map[string]any{
			FieldSuccess:  log.Success,
			FieldPayload:  log.Payload,
			FieldErrorMsg: log.ErrorMsg,
		},
		Time: pointTime(log.Time),
	}
	return s.store.EnqueueLog(ctx, point)
}

// WriteCommandRecord writes one command request record.
func (s *Service) WriteCommandRecord(ctx context.Context, modelKey string, record CommandRecord) error {
	if s.store == nil {
		return ErrDisabled
	}
	point := &Point{
		Measurement: CommandRecordTable(modelKey),
		Tags: map[string]string{
			TagModelID:   record.ModelID,
			TagDeviceID:  record.DeviceID,
			TagDeviceKey: record.DeviceKey,
		},
		Fields: map[string]any{
			FieldSuccess:             record.Success,
			FieldCommandPayload:      record.CommandPayload,
			FieldCommandReplyPayload: record.CommandReplyPayload,
			FieldMessageID:           record.MessageID,
		},
		Time: pointTime(record.Time),
	}
	return s.store.EnqueueData(ctx, point)
}

func pointTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now()
	}
	return t
}

func normalizeFields(values map[string]any) map[string]any {
	fields := make(map[string]any, len(values))
	for key, value := range values {
		normalized := normalizeValue(value)
		if normalized == nil {
			continue
		}
		fields[key] = normalized
	}
	return fields
}

func normalizeValue(value any) any {
	switch v := value.(type) {
	case nil:
		return nil
	case int:
		return int64(v)
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case uint:
		return uint64(v)
	case uint8:
		return uint64(v)
	case uint16:
		return uint64(v)
	case uint32:
		return uint64(v)
	case uint64:
		return v
	case float32:
		return float64(v)
	case float64:
		return v
	case string:
		return v
	case bool:
		return v
	case json.RawMessage:
		decoder := json.NewDecoder(bytes.NewReader(v))
		decoder.UseNumber()
		var decoded any
		if err := decoder.Decode(&decoded); err != nil {
			return string(v)
		}
		return normalizeValue(decoded)
	case json.Number:
		if integer, err := v.Int64(); err == nil {
			return integer
		}
		if floating, err := v.Float64(); err == nil {
			return floating
		}
		return v.String()
	case map[string]any, []any:
		encoded, err := json.Marshal(v)
		if err != nil {
			logging.V(2).InfoS("Failed to encode timeseries field", "err", err)
			return nil
		}
		return string(encoded)
	default:
		return v
	}
}
