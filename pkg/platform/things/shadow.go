package things

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"harns-platform/pkg/logging"
)

const (
	shadowFieldReported    = "reported"
	shadowFieldDesired     = "desired"
	shadowFieldVersion     = "version"
	shadowFieldTimestamp   = "timestamp"
	shadowLegacyReportTime = "lastReportTime"
	shadowUpdateRetries    = 3
)

type thingShadow struct {
	Reported  map[string]json.RawMessage
	Desired   map[string]json.RawMessage
	Version   int64
	Timestamp int64
}

type shadowUpdateGroup struct {
	timestamp int64
	values    map[string]any
}

// UpdateReported merges property values into the device shadow. Only updates
// newer than the shadow timestamp are applied.
func (s *Service) UpdateReported(ctx context.Context, thingID string, updates []ShadowUpdate) error {
	defer logging.Trace("things.UpdateReported", "thingId", thingID, "updates", len(updates))()

	if s.redis == nil || thingID == "" || len(updates) == 0 {
		return nil
	}

	key := shadowKeyPrefix + thingID
	now := time.Now().UnixMilli()
	var lastErr error
	for attempt := 0; attempt < shadowUpdateRetries; attempt++ {
		err := s.redis.Watch(ctx, func(tx *redis.Tx) error {
			values, err := tx.HMGet(ctx, key,
				shadowFieldReported, shadowFieldDesired, shadowFieldVersion, shadowFieldTimestamp).Result()
			if err != nil {
				return err
			}
			shadow, err := decodeThingShadow(values)
			if err != nil {
				return err
			}
			changed, err := applyShadowUpdates(shadow, updates, now)
			if err != nil {
				return err
			}
			if !changed {
				return nil
			}

			reported, err := json.Marshal(shadow.Reported)
			if err != nil {
				return err
			}
			desired, err := json.Marshal(shadow.Desired)
			if err != nil {
				return err
			}
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.HSet(ctx, key,
					shadowFieldReported, string(reported),
					shadowFieldDesired, string(desired),
					shadowFieldVersion, shadow.Version,
					shadowFieldTimestamp, shadow.Timestamp,
				)
				pipe.HDel(ctx, key, shadowLegacyReportTime)
				return nil
			})
			return err
		}, key)
		if errors.Is(err, redis.TxFailedErr) {
			lastErr = err
			continue
		}
		return err
	}
	if lastErr != nil {
		return fmt.Errorf("update thing shadow: %w", lastErr)
	}
	return errors.New("update thing shadow: too many retries")
}

func decodeThingShadow(values []interface{}) (*thingShadow, error) {
	shadow := &thingShadow{
		Reported: make(map[string]json.RawMessage),
		Desired:  make(map[string]json.RawMessage),
	}
	if len(values) > 0 {
		if raw := redisValue(values[0]); raw != "" {
			if err := json.Unmarshal([]byte(raw), &shadow.Reported); err != nil {
				return nil, fmt.Errorf("decode shadow reported: %w", err)
			}
			if shadow.Reported == nil {
				shadow.Reported = make(map[string]json.RawMessage)
			}
		}
	}
	if len(values) > 1 {
		if raw := redisValue(values[1]); raw != "" {
			if err := json.Unmarshal([]byte(raw), &shadow.Desired); err != nil {
				return nil, fmt.Errorf("decode shadow desired: %w", err)
			}
			if shadow.Desired == nil {
				shadow.Desired = make(map[string]json.RawMessage)
			}
		}
	}
	if len(values) > 2 {
		shadow.Version, _ = strconv.ParseInt(redisValue(values[2]), 10, 64)
	}
	if len(values) > 3 {
		shadow.Timestamp, _ = strconv.ParseInt(redisValue(values[3]), 10, 64)
	}
	return shadow, nil
}

func applyShadowUpdates(shadow *thingShadow, updates []ShadowUpdate, now int64) (bool, error) {
	if shadow == nil {
		return false, nil
	}
	changed := false
	for _, group := range groupShadowUpdates(updates, now) {
		if group.timestamp <= shadow.Timestamp {
			continue
		}
		for identifier, value := range group.values {
			raw, err := json.Marshal(value)
			if err != nil {
				return false, fmt.Errorf("encode shadow property %s: %w", identifier, err)
			}
			shadow.Reported[identifier] = raw
		}
		shadow.Timestamp = group.timestamp
		changed = true
	}
	if !changed {
		return false, nil
	}

	for identifier, desired := range shadow.Desired {
		reported, ok := shadow.Reported[identifier]
		if !ok {
			continue
		}
		equal, err := jsonValuesEqual(reported, desired)
		if err != nil {
			logging.V(2).InfoS("Failed to compare shadow property", "identifier", identifier, "err", err)
			continue
		}
		if equal {
			delete(shadow.Desired, identifier)
			changed = true
		}
	}
	if changed {
		shadow.Version++
	}
	return changed, nil
}

func groupShadowUpdates(updates []ShadowUpdate, now int64) []shadowUpdateGroup {
	groups := make([]shadowUpdateGroup, 0, len(updates))
	index := make(map[int64]int, len(updates))
	for _, update := range updates {
		if len(update.Values) == 0 {
			continue
		}
		timestamp := update.Time.UnixMilli()
		if timestamp <= 0 {
			timestamp = now
		}
		if i, ok := index[timestamp]; ok {
			for identifier, value := range update.Values {
				groups[i].values[identifier] = value
			}
			continue
		}
		values := make(map[string]any, len(update.Values))
		for identifier, value := range update.Values {
			values[identifier] = value
		}
		index[timestamp] = len(groups)
		groups = append(groups, shadowUpdateGroup{timestamp: timestamp, values: values})
	}
	sort.SliceStable(groups, func(i, j int) bool {
		return groups[i].timestamp < groups[j].timestamp
	})
	return groups
}

func jsonValuesEqual(left, right json.RawMessage) (bool, error) {
	leftCanonical, err := canonicalJSON(left)
	if err != nil {
		return false, err
	}
	rightCanonical, err := canonicalJSON(right)
	if err != nil {
		return false, err
	}
	return bytes.Equal(leftCanonical, rightCanonical), nil
}

func canonicalJSON(raw json.RawMessage) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return json.Marshal(normalizeJSONValue(value))
}

func normalizeJSONValue(value any) any {
	switch typed := value.(type) {
	case json.Number:
		if integer, err := typed.Int64(); err == nil {
			return integer
		}
		if floating, err := typed.Float64(); err == nil {
			return floating
		}
		return typed.String()
	case map[string]any:
		for key, item := range typed {
			typed[key] = normalizeJSONValue(item)
		}
		return typed
	case []any:
		for i := range typed {
			typed[i] = normalizeJSONValue(typed[i])
		}
		return typed
	default:
		return value
	}
}

func redisValue(value interface{}) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return fmt.Sprint(typed)
	}
}
