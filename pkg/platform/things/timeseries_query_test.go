package things

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/platform/thingmodel/tsl"
	"harns-platform/pkg/platform/things/v1"
)

func TestFlexibleTimeUnmarshal(t *testing.T) {
	var millis v1.FlexibleTime
	require.NoError(t, millis.UnmarshalJSON([]byte(`1773827321000`)))
	assert.Equal(t, int64(1773827321000), millis.UnixMilli())

	var text v1.FlexibleTime
	require.NoError(t, text.UnmarshalJSON([]byte(`"2026-11-05T10:00:00Z"`)))
	assert.Equal(t, time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC), text.Time)
}

func TestMergeTimeSeriesItems(t *testing.T) {
	items, issues := mergeTimeSeriesItems([]v1.ThingTimeSeriesItem{
		{ThingID: "thing-1", Properties: []string{"a", "b"}},
		{ThingID: "thing-1", Properties: []string{"b", "c"}},
		{ThingID: "thing-2", Properties: []string{"d"}},
	})
	require.Empty(t, issues)
	require.Len(t, items, 2)
	assert.Equal(t, []string{"a", "b", "c"}, items[0].Properties)
	assert.Equal(t, []string{"d"}, items[1].Properties)
}

func supportsDefaultAggregate(name string) bool {
	switch name {
	case "raw", "min", "max", "avg", "count", "sum", "median", "stddev":
		return true
	default:
		return false
	}
}

func TestNormalizeTimeSeriesRequest(t *testing.T) {
	start := time.Now()
	req := v1.ThingTimeSeriesReq{
		StartTime: v1.FlexibleTime{Time: start},
		EndTime:   v1.FlexibleTime{Time: start.Add(time.Hour)},
	}
	aggregate, sort, limit, interval, err := normalizeTimeSeriesRequest(req, supportsDefaultAggregate)
	require.NoError(t, err)
	assert.Equal(t, "raw", aggregate)
	assert.Equal(t, "asc", sort)
	assert.Equal(t, 100, limit)
	assert.Empty(t, interval)
}

func TestNormalizeTimeSeriesRequestRejectsRawInterval(t *testing.T) {
	start := time.Now()
	req := v1.ThingTimeSeriesReq{
		StartTime: v1.FlexibleTime{Time: start},
		EndTime:   v1.FlexibleTime{Time: start.Add(time.Hour)},
		Aggregate: "raw",
		Interval:  "5m",
	}
	_, _, _, _, err := normalizeTimeSeriesRequest(req, supportsDefaultAggregate)
	require.Error(t, err)
}

func TestAggregateNeedsNumeric(t *testing.T) {
	assert.False(t, aggregateNeedsNumeric("raw"))
	assert.False(t, aggregateNeedsNumeric("count"))
	assert.True(t, aggregateNeedsNumeric("sum"))
	assert.True(t, aggregateNeedsNumeric("median"))
}

func TestNumericDataType(t *testing.T) {
	assert.True(t, numericDataType(tsl.DataTypeInt))
	assert.True(t, numericDataType(tsl.DataTypeDouble))
	assert.False(t, numericDataType(tsl.DataTypeBool))
	assert.False(t, numericDataType(tsl.DataTypeText))
}
