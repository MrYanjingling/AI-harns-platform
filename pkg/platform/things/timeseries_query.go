package things

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/logging"
	"harns-platform/pkg/platform/thingmodel/tsl"
	"harns-platform/pkg/platform/things/v1"
	"harns-platform/pkg/timeseries"
)

// ThingTimeSeriesQuerier executes model-level property series queries.
type ThingTimeSeriesQuerier interface {
	QueryPropertySeries(ctx context.Context, query timeseries.PropertySeriesQuery) ([]timeseries.PropertySeries, error)
	SupportsAggregate(name string) bool
}

type resolvedTimeSeriesThing struct {
	ThingID    string
	ModelKey   string
	Debug      bool
	Properties []string
}

// GetThingTimeSeries queries one or more things across their own thing models.
func (s *Service) GetThingTimeSeries(ctx context.Context, req v1.ThingTimeSeriesReq) (*v1.ThingTimeSeriesVo, error) {
	defer logging.Trace("things.GetThingTimeSeries", "things", len(req.Things))()

	if s.models == nil {
		return nil, response.ErrInternal(errors.New("thing model resolver is not configured"))
	}
	if s.timeSeries == nil {
		return nil, response.ErrInternal(errors.New("timeseries querier is not configured"))
	}
	aggregate, sort, limit, interval, err := normalizeTimeSeriesRequest(req, s.timeSeries.SupportsAggregate)
	if err != nil {
		return nil, err
	}

	items, issues := mergeTimeSeriesItems(req.Things)
	if verr := NewValidationError(issues); verr != nil {
		return nil, verr
	}

	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ThingID)
	}
	snapshots, err := s.GetThingSnapshots(ctx, ids)
	if err != nil {
		return nil, response.ErrInternal(err)
	}

	resolved := make([]resolvedTimeSeriesThing, 0, len(items))
	queryErrors := make([]v1.ThingTimeSeriesErrorVo, 0)
	for _, item := range items {
		snapshot, ok := snapshots[item.ThingID]
		if !ok {
			queryErrors = append(queryErrors, v1.ThingTimeSeriesErrorVo{
				ThingID: item.ThingID,
				Message: "thing not found",
			})
			continue
		}
		modelKey := snapshot.ThingModelKey
		definitions, err := s.models.GetPropertiesByKey(ctx, modelKey)
		if err != nil {
			queryErrors = append(queryErrors, v1.ThingTimeSeriesErrorVo{
				ThingID: item.ThingID,
				Message: "load thing model properties failed",
			})
			continue
		}

		properties := make([]string, 0, len(item.Properties))
		for _, identifier := range item.Properties {
			definition, ok := definitions[identifier]
			if !ok {
				queryErrors = append(queryErrors, v1.ThingTimeSeriesErrorVo{
					ThingID:  item.ThingID,
					Property: identifier,
					Message:  "property not found",
				})
				continue
			}
			if aggregateNeedsNumeric(aggregate) && !numericDataType(definition.DataType.Type) {
				queryErrors = append(queryErrors, v1.ThingTimeSeriesErrorVo{
					ThingID:  item.ThingID,
					Property: identifier,
					Message:  "aggregate requires a numeric property",
				})
				continue
			}
			properties = append(properties, identifier)
		}
		if len(properties) == 0 {
			continue
		}
		resolved = append(resolved, resolvedTimeSeriesThing{
			ThingID:    item.ThingID,
			ModelKey:   modelKey,
			Debug:      snapshot.Debug == 1,
			Properties: uniqueStrings(properties),
		})
	}

	type groupKey struct {
		Debug    bool
		ModelKey string
	}
	groups := make(map[groupKey][]resolvedTimeSeriesThing)
	for _, item := range resolved {
		key := groupKey{Debug: item.Debug, ModelKey: item.ModelKey}
		groups[key] = append(groups[key], item)
	}

	seriesByThing := make(map[string]v1.ThingTimeSeriesSeriesVo, len(resolved))
	for key, group := range groups {
		modelKey := key.ModelKey
		deviceIDs := make([]string, 0, len(group))
		properties := make([]string, 0)
		for _, item := range group {
			deviceIDs = append(deviceIDs, item.ThingID)
			properties = append(properties, item.Properties...)
		}
		properties = uniqueStrings(properties)

		series, err := s.timeSeries.QueryPropertySeries(ctx, timeseries.PropertySeriesQuery{
			ModelKey:   modelKey,
			Debug:      key.Debug,
			DeviceIDs:  deviceIDs,
			Properties: properties,
			StartTime:  req.StartTime.Time,
			EndTime:    req.EndTime.Time,
			Aggregate:  aggregate,
			Interval:   interval,
			Sort:       sort,
			Limit:      limit,
		})
		if err != nil {
			logging.ErrorS(err, "Failed to query thing time series", "thingModelKey", modelKey, "devices", len(deviceIDs))
			return nil, response.ErrInternal(err)
		}

		byDevice := make(map[string]timeseries.PropertySeries, len(series))
		for _, item := range series {
			byDevice[item.DeviceID] = item
		}
		global := aggregate != "raw" && interval == ""
		for _, item := range group {
			result := v1.ThingTimeSeriesSeriesVo{ThingID: item.ThingID, ModelKey: modelKey}
			data := byDevice[item.ThingID]
			if global {
				result.Values = selectValues(data.Values, item.Properties)
			} else {
				result.Points = make([]v1.ThingTimeSeriesPointVo, 0, len(data.Points))
				for _, point := range data.Points {
					result.Points = append(result.Points, v1.ThingTimeSeriesPointVo{
						Time:   point.Time,
						Values: selectValues(point.Values, item.Properties),
					})
				}
			}
			seriesByThing[item.ThingID] = result
		}
	}

	out := &v1.ThingTimeSeriesVo{
		Aggregate: aggregate,
		Interval:  interval,
		Series:    make([]v1.ThingTimeSeriesSeriesVo, 0, len(resolved)),
		Errors:    queryErrors,
	}
	for _, item := range items {
		if result, ok := seriesByThing[item.ThingID]; ok {
			out.Series = append(out.Series, result)
		}
	}
	return out, nil
}

func normalizeTimeSeriesRequest(req v1.ThingTimeSeriesReq, supportsAggregate func(string) bool) (string, string, int, string, error) {
	issues := make([]FieldIssue, 0)
	aggregate := strings.ToLower(strings.TrimSpace(req.Aggregate))
	if aggregate == "" {
		aggregate = "raw"
	}
	if supportsAggregate == nil || !supportsAggregate(aggregate) {
		issues = append(issues, FieldIssue{Field: "aggregate", MsgID: "validate.thing_timeseries_aggregate_invalid"})
	}

	sort := strings.ToLower(strings.TrimSpace(req.Sort))
	if sort == "" {
		sort = "asc"
	}
	if sort != "asc" && sort != "desc" {
		issues = append(issues, FieldIssue{Field: "sort", MsgID: "validate.thing_timeseries_sort_invalid"})
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 2000 {
		issues = append(issues, FieldIssue{Field: "limit", MsgID: "validate.thing_timeseries_limit_invalid"})
	}

	interval := strings.TrimSpace(req.Interval)
	if interval != "" {
		if aggregate == "raw" {
			issues = append(issues, FieldIssue{Field: "interval", MsgID: "validate.thing_timeseries_interval_invalid"})
		} else if err := timeseries.ValidateInterval(interval); err != nil {
			issues = append(issues, FieldIssue{Field: "interval", MsgID: "validate.thing_timeseries_interval_invalid"})
		}
	}
	if start := req.StartTime.Time; start.IsZero() || start.UnixMilli() <= 0 ||
		req.EndTime.Time.IsZero() || req.EndTime.Time.UnixMilli() <= 0 ||
		!start.Before(req.EndTime.Time) {
		issues = append(issues, FieldIssue{Field: "endTime", MsgID: "validate.thing_timeseries_time_invalid"})
	}
	if verr := NewValidationError(issues); verr != nil {
		return "", "", 0, "", verr
	}
	return aggregate, sort, limit, interval, nil
}

func mergeTimeSeriesItems(items []v1.ThingTimeSeriesItem) ([]v1.ThingTimeSeriesItem, []FieldIssue) {
	merged := make([]v1.ThingTimeSeriesItem, 0, len(items))
	index := make(map[string]int, len(items))
	issues := make([]FieldIssue, 0)

	for i, item := range items {
		thingID := strings.TrimSpace(item.ThingID)
		if thingID == "" {
			issues = append(issues, FieldIssue{Field: fmt.Sprintf("things[%d].thingId", i), MsgID: "validate.thing_timeseries_thing_required"})
			continue
		}
		properties := make([]string, 0, len(item.Properties))
		for j, property := range item.Properties {
			property = strings.TrimSpace(property)
			if property == "" {
				issues = append(issues, FieldIssue{Field: fmt.Sprintf("things[%d].properties[%d]", i, j), MsgID: "validate.thing_timeseries_properties_required"})
				continue
			}
			properties = append(properties, property)
		}
		if len(properties) == 0 {
			issues = append(issues, FieldIssue{Field: fmt.Sprintf("things[%d].properties", i), MsgID: "validate.thing_timeseries_properties_required"})
			continue
		}
		if pos, ok := index[thingID]; ok {
			merged[pos].Properties = append(merged[pos].Properties, properties...)
			continue
		}
		index[thingID] = len(merged)
		merged = append(merged, v1.ThingTimeSeriesItem{ThingID: thingID, Properties: properties})
	}
	for i := range merged {
		merged[i].Properties = uniqueStrings(merged[i].Properties)
	}
	return merged, issues
}

func selectValues(values map[string]any, properties []string) map[string]any {
	out := make(map[string]any, len(properties))
	for _, property := range properties {
		out[property] = values[property]
	}
	return out
}

func aggregateNeedsNumeric(aggregate string) bool {
	return aggregate != "raw" && aggregate != "count"
}

func numericDataType(dataType tsl.DataType) bool {
	switch dataType {
	case tsl.DataTypeInt, tsl.DataTypeLong, tsl.DataTypeFloat, tsl.DataTypeDouble:
		return true
	default:
		return false
	}
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
