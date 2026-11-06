package timeseries

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPropertyQueryLimit = 100
	maxPropertyQueryLimit     = 2000
)

var (
	intervalPattern      = regexp.MustCompile(`^([0-9]+)(ns|us|µs|ms|s|m|h|d|w)$`)
	aggregateNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

// PropertySeriesQuery is one model-level multi-device time series query.
type PropertySeriesQuery struct {
	ModelKey   string
	Debug      bool
	DeviceIDs  []string
	Properties []string
	StartTime  time.Time
	EndTime    time.Time
	Aggregate  string
	Interval   string
	Sort       string
	Limit      int
}

// PropertySeries is one device's query result.
type PropertySeries struct {
	DeviceID string
	Points   []PropertySeriesPoint
	Values   map[string]any
}

// PropertySeriesPoint is one time series point.
type PropertySeriesPoint struct {
	Time   int64
	Values map[string]any
}

// QueryPropertySeries executes a property time series query.
func (s *Service) QueryPropertySeries(ctx context.Context, q PropertySeriesQuery) ([]PropertySeries, error) {
	if s.store == nil {
		return nil, ErrDisabled
	}
	normalized, intervalSQL, err := normalizePropertySeriesQuery(q, s.SupportsAggregate)
	if err != nil {
		return nil, err
	}

	database := s.cfg.Database
	if normalized.Debug && s.cfg.DebugDatabase != "" {
		database = s.cfg.DebugDatabase
	}
	columns, err := s.store.QueryColumnNames(ctx, database, PropertyTable(normalized.ModelKey))
	if err != nil {
		return nil, err
	}
	existing := make([]string, 0, len(normalized.Properties))
	for _, property := range normalized.Properties {
		if _, ok := columns[property]; ok {
			existing = append(existing, property)
		}
	}
	if len(existing) == 0 {
		return emptyPropertySeries(normalized), nil
	}

	query, err := buildPropertySeriesSQL(normalized, intervalSQL, existing)
	if err != nil {
		return nil, err
	}
	rows, err := s.store.QuerySQL(ctx, database, query)
	if err != nil {
		return nil, err
	}
	return parsePropertySeriesRows(normalized, rows)
}

// SupportsAggregate reports whether the aggregate is enabled by configuration.
func (s *Service) SupportsAggregate(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "raw" {
		return true
	}
	if !aggregateNamePattern.MatchString(name) || s == nil {
		return false
	}
	aggregates := s.cfg.QueryAggregates
	if len(aggregates) == 0 {
		aggregates = DefaultConfig().QueryAggregates
	}
	for _, aggregate := range aggregates {
		if strings.EqualFold(strings.TrimSpace(aggregate), name) {
			return true
		}
	}
	return false
}

func normalizePropertySeriesQuery(q PropertySeriesQuery, supportsAggregate func(string) bool) (PropertySeriesQuery, string, error) {
	q.Aggregate = strings.ToLower(strings.TrimSpace(q.Aggregate))
	if q.Aggregate == "" {
		q.Aggregate = "raw"
	}
	if supportsAggregate == nil || !supportsAggregate(q.Aggregate) {
		return q, "", fmt.Errorf("invalid aggregate %q", q.Aggregate)
	}

	q.Sort = strings.ToLower(strings.TrimSpace(q.Sort))
	if q.Sort == "" {
		q.Sort = "asc"
	}
	if q.Sort != "asc" && q.Sort != "desc" {
		return q, "", fmt.Errorf("invalid sort %q", q.Sort)
	}

	if q.Limit <= 0 {
		q.Limit = defaultPropertyQueryLimit
	}
	if q.Limit > maxPropertyQueryLimit {
		q.Limit = maxPropertyQueryLimit
	}
	if q.StartTime.IsZero() || q.EndTime.IsZero() || !q.StartTime.Before(q.EndTime) {
		return q, "", fmt.Errorf("invalid time range")
	}
	if len(q.DeviceIDs) == 0 {
		return q, "", fmt.Errorf("device ids are required")
	}
	if len(q.Properties) == 0 {
		return q, "", fmt.Errorf("properties are required")
	}

	q.Interval = strings.TrimSpace(q.Interval)
	intervalSQL := ""
	if q.Interval != "" {
		if q.Aggregate == "raw" {
			return q, "", fmt.Errorf("interval cannot be used with aggregate raw")
		}
		var err error
		intervalSQL, err = parseInterval(q.Interval)
		if err != nil {
			return q, "", err
		}
	}
	return q, intervalSQL, nil
}

// ValidateInterval reports whether the interval format is supported.
func ValidateInterval(value string) error {
	_, err := parseInterval(value)
	return err
}

func parseInterval(value string) (string, error) {
	matches := intervalPattern.FindStringSubmatch(strings.ToLower(strings.TrimSpace(value)))
	if matches == nil {
		return "", fmt.Errorf("invalid interval %q", value)
	}
	count, err := strconv.Atoi(matches[1])
	if err != nil || count <= 0 {
		return "", fmt.Errorf("invalid interval %q", value)
	}
	unit := matches[2]
	names := map[string]string{
		"ns": "nanoseconds",
		"us": "microseconds",
		"µs": "microseconds",
		"ms": "milliseconds",
		"s":  "seconds",
		"m":  "minutes",
		"h":  "hours",
		"d":  "days",
		"w":  "weeks",
	}
	return fmt.Sprintf("%d %s", count, names[unit]), nil
}

func buildPropertySeriesSQL(q PropertySeriesQuery, intervalSQL string, properties []string) (string, error) {
	if len(properties) == 0 {
		return "", fmt.Errorf("properties are required")
	}
	table := quoteIdentifier(PropertyTable(q.ModelKey))
	devices := make([]string, 0, len(q.DeviceIDs))
	for _, id := range q.DeviceIDs {
		devices = append(devices, quoteLiteral(id))
	}
	where := fmt.Sprintf("device_id IN (%s) AND time >= TIMESTAMP '%s' AND time <= TIMESTAMP '%s'",
		strings.Join(devices, ", "), formatTimestamp(q.StartTime), formatTimestamp(q.EndTime))
	order := strings.ToUpper(q.Sort)
	propertyList := quoteIdentifiers(properties)

	if q.Aggregate != "raw" && q.Interval == "" {
		aggregates := make([]string, 0, len(properties))
		for _, property := range properties {
			aggregates = append(aggregates, fmt.Sprintf("%s(%s) AS %s", q.Aggregate, quoteIdentifier(property), quoteIdentifier(property)))
		}
		return fmt.Sprintf(`SELECT device_id, %s FROM %s WHERE %s GROUP BY device_id`,
			strings.Join(aggregates, ", "), table, where), nil
	}

	if q.Aggregate == "raw" {
		return fmt.Sprintf(`SELECT time, device_id, %s
FROM (
  SELECT time, device_id, %s,
         ROW_NUMBER() OVER (PARTITION BY device_id ORDER BY time %s) AS __rn
  FROM %s
  WHERE %s
) q
WHERE __rn <= %d
ORDER BY time %s`, propertyList, propertyList, order, table, where, q.Limit, order), nil
	}

	aggregates := make([]string, 0, len(properties))
	for _, property := range properties {
		aggregates = append(aggregates, fmt.Sprintf("%s(%s) AS %s", q.Aggregate, quoteIdentifier(property), quoteIdentifier(property)))
	}
	bucket := fmt.Sprintf("date_bin(INTERVAL '%s', time, TIMESTAMP '%s')", intervalSQL, formatTimestamp(q.StartTime))
	return fmt.Sprintf(`SELECT time, device_id, %s
FROM (
  SELECT time, device_id, %s,
         ROW_NUMBER() OVER (PARTITION BY device_id ORDER BY time %s) AS __rn
  FROM (
    SELECT %s AS time, device_id, %s
    FROM %s
    WHERE %s
    GROUP BY 1, device_id
  ) buckets
) q
WHERE __rn <= %d
ORDER BY time %s`, propertyList, propertyList, order, bucket, strings.Join(aggregates, ", "), table, where, q.Limit, order), nil
}

func parsePropertySeriesRows(q PropertySeriesQuery, rows []map[string]any) ([]PropertySeries, error) {
	global := q.Aggregate != "raw" && q.Interval == ""
	byDevice := make(map[string]*PropertySeries, len(q.DeviceIDs))
	series := make([]PropertySeries, 0, len(q.DeviceIDs))
	for _, deviceID := range q.DeviceIDs {
		item := PropertySeries{DeviceID: deviceID}
		if global {
			item.Values = emptyValues(q.Properties, q.Aggregate)
		}
		byDevice[deviceID] = &item
	}

	for _, row := range rows {
		deviceID, _ := row["device_id"].(string)
		item, ok := byDevice[deviceID]
		if !ok {
			continue
		}
		values := emptyValues(q.Properties, q.Aggregate)
		for _, property := range q.Properties {
			if value, exists := row[property]; exists {
				values[property] = value
			}
		}
		if global {
			item.Values = values
			continue
		}
		millis, err := parseSeriesTime(row["time"])
		if err != nil {
			return nil, err
		}
		item.Points = append(item.Points, PropertySeriesPoint{Time: millis, Values: values})
	}

	for _, deviceID := range q.DeviceIDs {
		series = append(series, *byDevice[deviceID])
	}
	return series, nil
}

func emptyPropertySeries(q PropertySeriesQuery) []PropertySeries {
	global := q.Aggregate != "raw" && q.Interval == ""
	series := make([]PropertySeries, 0, len(q.DeviceIDs))
	for _, deviceID := range q.DeviceIDs {
		item := PropertySeries{DeviceID: deviceID}
		if global {
			item.Values = emptyValues(q.Properties, q.Aggregate)
		}
		series = append(series, item)
	}
	return series
}

func emptyValues(properties []string, aggregate string) map[string]any {
	values := make(map[string]any, len(properties))
	for _, property := range properties {
		if aggregate == "count" {
			values[property] = int64(0)
		} else {
			values[property] = nil
		}
	}
	return values
}

func parseSeriesTime(value any) (int64, error) {
	switch typed := value.(type) {
	case string:
		parsed, err := time.Parse(time.RFC3339Nano, typed)
		if err != nil {
			return 0, fmt.Errorf("invalid time %q: %w", typed, err)
		}
		return parsed.UnixMilli(), nil
	case json.Number:
		millis, err := typed.Int64()
		if err != nil {
			return 0, fmt.Errorf("invalid time %q: %w", typed, err)
		}
		return millis, nil
	case float64:
		return int64(typed), nil
	case int64:
		return typed, nil
	case int:
		return int64(typed), nil
	default:
		return 0, fmt.Errorf("invalid time value %v", value)
	}
}

func formatTimestamp(value time.Time) string {
	return value.UTC().Format("2006-01-02T15:04:05.000Z")
}

func quoteIdentifiers(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, quoteIdentifier(value))
	}
	return strings.Join(quoted, ", ")
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func quoteLiteral(value string) string {
	return `'` + strings.ReplaceAll(value, `'`, `''`) + `'`
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
