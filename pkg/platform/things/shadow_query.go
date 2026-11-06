package things

import (
	"context"
	"encoding/json"

	"harns-platform/pkg/apis"
	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/logging"
	"harns-platform/pkg/platform/thingmodel/tsl"
	"harns-platform/pkg/platform/things/v1"
)

// GetThingShadowData returns paginated model property definitions merged with shadow values.
func (s *Service) GetThingShadowData(ctx context.Context, thingID string, q v1.ThingShadowQuery) (*v1.ThingShadowVo, error) {
	defer logging.Trace("things.GetThingShadowData", "thingId", thingID, "keyword", q.Keyword, "page", q.Page, "pageSize", q.PageSize)()

	q.Normalize()
	item, err := s.getEntity(ctx, thingID)
	if err != nil {
		return nil, err
	}
	properties, total, err := s.store.ListThingModelProperties(ctx, item.ThingModelID, q.PageQuery, q.Keyword)
	if err != nil {
		return nil, response.ErrInternal(err)
	}

	shadow := &thingShadow{
		Reported: make(map[string]json.RawMessage),
		Desired:  make(map[string]json.RawMessage),
	}
	if s.redis != nil {
		values, err := s.redis.HMGet(ctx, shadowKeyPrefix+thingID,
			shadowFieldReported, shadowFieldDesired, shadowFieldVersion, shadowFieldTimestamp).Result()
		if err != nil {
			logging.ErrorS(err, "Failed to read thing shadow", "thingId", thingID)
		} else if decoded, err := decodeThingShadow(values); err != nil {
			logging.ErrorS(err, "Failed to decode thing shadow", "thingId", thingID)
		} else {
			shadow = decoded
		}
	}

	return &v1.ThingShadowVo{
		ThingID:    item.ID,
		Version:    shadow.Version,
		Timestamp:  shadow.Timestamp,
		Properties: *apis.NewPageResp(buildShadowProperties(properties, shadow.Reported, shadow.Desired), total, q.PageQuery),
	}, nil
}

func buildShadowProperties(properties []tsl.Property, reported, desired map[string]json.RawMessage) []v1.ThingShadowPropertyVo {
	out := make([]v1.ThingShadowPropertyVo, 0, len(properties))
	for _, property := range properties {
		out = append(out, v1.ThingShadowPropertyVo{
			Identifier:   property.Identifier,
			Name:         property.Name,
			DataType:     property.DataType,
			RWFlag:       property.RWFlag,
			Required:     property.Required,
			Description:  property.Description,
			Custom:       property.Custom,
			Sort:         property.Sort,
			Value:        reported[property.Identifier],
			DesiredValue: desired[property.Identifier],
		})
	}
	return out
}
