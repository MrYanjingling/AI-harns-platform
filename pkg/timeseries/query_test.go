package timeseries

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInterval(t *testing.T) {
	got, err := parseInterval("5m")
	require.NoError(t, err)
	assert.Equal(t, "5 minutes", got)

	got, err = parseInterval("1d")
	require.NoError(t, err)
	assert.Equal(t, "1 days", got)

	_, err = parseInterval("1month")
	require.Error(t, err)
}

func TestBuildPropertySeriesSQLRaw(t *testing.T) {
	start := time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC)
	q := PropertySeriesQuery{
		ModelKey:   "tm_demo",
		DeviceIDs:  []string{"thing-1", "thing-2"},
		Properties: []string{"temperature", "humidity"},
		StartTime:  start,
		EndTime:    start.Add(time.Hour),
		Aggregate:  "raw",
		Sort:       "desc",
		Limit:      100,
	}
	normalized, intervalSQL, err := normalizePropertySeriesQuery(q, func(name string) bool {
		return name == "raw" || name == "avg" || name == "min" || name == "max" || name == "sum" || name == "count" || name == "median" || name == "stddev"
	})
	require.NoError(t, err)
	sql, err := buildPropertySeriesSQL(normalized, intervalSQL, normalized.Properties)
	require.NoError(t, err)
	assert.Contains(t, sql, `FROM "device_property_tm_demo"`)
	assert.Contains(t, sql, `"temperature"`)
	assert.Contains(t, sql, `ROW_NUMBER() OVER (PARTITION BY device_id ORDER BY time DESC)`)
	assert.Contains(t, sql, `WHERE __rn <= 100`)
	assert.Contains(t, sql, `ORDER BY time DESC`)
}

func TestBuildPropertySeriesSQLGlobalAggregate(t *testing.T) {
	start := time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC)
	q := PropertySeriesQuery{
		ModelKey:   "tm_demo",
		DeviceIDs:  []string{"thing-1"},
		Properties: []string{"temperature"},
		StartTime:  start,
		EndTime:    start.Add(time.Hour),
		Aggregate:  "avg",
	}
	normalized, intervalSQL, err := normalizePropertySeriesQuery(q, func(name string) bool {
		return name == "raw" || name == "avg" || name == "min" || name == "max" || name == "sum" || name == "count" || name == "median" || name == "stddev"
	})
	require.NoError(t, err)
	sql, err := buildPropertySeriesSQL(normalized, intervalSQL, normalized.Properties)
	require.NoError(t, err)
	assert.Contains(t, sql, `avg("temperature") AS "temperature"`)
	assert.Contains(t, sql, `GROUP BY device_id`)
	assert.NotContains(t, sql, `ROW_NUMBER`)
}

func TestBuildPropertySeriesSQLInterval(t *testing.T) {
	start := time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC)
	q := PropertySeriesQuery{
		ModelKey:   "tm_demo",
		DeviceIDs:  []string{"thing-1"},
		Properties: []string{"temperature"},
		StartTime:  start,
		EndTime:    start.Add(time.Hour),
		Aggregate:  "avg",
		Interval:   "5m",
		Limit:      10,
	}
	normalized, intervalSQL, err := normalizePropertySeriesQuery(q, func(name string) bool {
		return name == "raw" || name == "avg" || name == "min" || name == "max" || name == "sum" || name == "count" || name == "median" || name == "stddev"
	})
	require.NoError(t, err)
	sql, err := buildPropertySeriesSQL(normalized, intervalSQL, normalized.Properties)
	require.NoError(t, err)
	assert.Contains(t, sql, `date_bin(INTERVAL '5 minutes'`)
	assert.Contains(t, sql, `GROUP BY 1, device_id`)
	assert.Contains(t, sql, `ROW_NUMBER()`)
}

func TestParsePropertySeriesRows(t *testing.T) {
	start := time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC)
	q := PropertySeriesQuery{
		DeviceIDs:  []string{"thing-1", "thing-2"},
		Properties: []string{"temperature", "humidity"},
		StartTime:  start,
		EndTime:    start.Add(time.Hour),
		Aggregate:  "raw",
	}
	rows := []map[string]any{
		{"time": "2026-11-05T10:00:00.123Z", "device_id": "thing-1", "temperature": json.Number("26.5")},
	}
	series, err := parsePropertySeriesRows(q, rows)
	require.NoError(t, err)
	require.Len(t, series, 2)
	assert.Equal(t, "thing-1", series[0].DeviceID)
	require.Len(t, series[0].Points, 1)
	expectedTime, err := time.Parse(time.RFC3339Nano, "2026-11-05T10:00:00.123Z")
	require.NoError(t, err)
	assert.Equal(t, expectedTime.UnixMilli(), series[0].Points[0].Time)
	assert.Equal(t, json.Number("26.5"), series[0].Points[0].Values["temperature"])
	assert.Nil(t, series[0].Points[0].Values["humidity"])
	assert.Empty(t, series[1].Points)
}

func TestParsePropertySeriesRowsGlobal(t *testing.T) {
	start := time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC)
	q := PropertySeriesQuery{
		DeviceIDs:  []string{"thing-1"},
		Properties: []string{"temperature"},
		StartTime:  start,
		EndTime:    start.Add(time.Hour),
		Aggregate:  "avg",
	}
	series, err := parsePropertySeriesRows(q, []map[string]any{
		{"device_id": "thing-1", "temperature": json.Number("26.5")},
	})
	require.NoError(t, err)
	require.Len(t, series, 1)
	assert.Equal(t, json.Number("26.5"), series[0].Values["temperature"])
}

func TestSupportsAggregate(t *testing.T) {
	svc := &Service{cfg: Config{QueryAggregates: []string{"min", "count"}}}
	assert.True(t, svc.SupportsAggregate("raw"))
	assert.True(t, svc.SupportsAggregate("MIN"))
	assert.True(t, svc.SupportsAggregate("count"))
	assert.False(t, svc.SupportsAggregate("avg"))
}

func TestCountEmptyValues(t *testing.T) {
	values := emptyValues([]string{"a", "b"}, "count")
	assert.Equal(t, int64(0), values["a"])
	assert.Equal(t, int64(0), values["b"])
}
