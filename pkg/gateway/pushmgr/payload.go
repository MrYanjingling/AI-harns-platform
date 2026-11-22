package pushmgr

import (
	"encoding/json"
	"sort"
	"strconv"
	"time"

	"harns-platform/pkg/gateway/collection"
)

type propertyMessage struct {
	Type    string         `json:"type"`
	Payload []propertyItem `json:"payload"`
}

type propertyItem struct {
	CollectionID string            `json:"collectionId"`
	Time         int64             `json:"time"`
	Property     map[string]string `json:"property"`
}

func buildPropertyMessages(snapshots []CollectionSnapshot, maxPerMessage int) []propertyMessage {
	now := time.Now().UnixMilli()
	sort.Slice(snapshots, func(i, j int) bool { return snapshots[i].CollectionID < snapshots[j].CollectionID })
	items := make([]propertyItem, 0, len(snapshots))
	for _, snapshot := range snapshots {
		if snapshot.Status != collection.CollectStatusCollecting {
			continue
		}
		values := make(map[string]string, len(snapshot.Values))
		names := make([]string, 0, len(snapshot.Values))
		for name := range snapshot.Values {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			value, ok := propertyValue(snapshot.Values[name])
			if ok {
				values[name] = value
			}
		}
		timestamp := snapshot.Time.UnixMilli()
		if snapshot.Time.IsZero() {
			timestamp = now
		}
		items = append(items, propertyItem{
			CollectionID: snapshot.CollectionID,
			Time:         timestamp,
			Property:     values,
		})
	}
	if len(items) == 0 {
		return nil
	}
	if maxPerMessage <= 0 {
		return []propertyMessage{{Type: "property", Payload: items}}
	}
	out := make([]propertyMessage, 0, (len(items)+maxPerMessage-1)/maxPerMessage)
	for start := 0; start < len(items); start += maxPerMessage {
		end := start + maxPerMessage
		if end > len(items) {
			end = len(items)
		}
		out = append(out, propertyMessage{Type: "property", Payload: items[start:end]})
	}
	return out
}

func propertyValue(value any) (string, bool) {
	switch v := value.(type) {
	case nil:
		return "", false
	case string:
		return v, true
	case bool:
		return strconv.FormatBool(v), true
	case int:
		return strconv.FormatInt(int64(v), 10), true
	case int8:
		return strconv.FormatInt(int64(v), 10), true
	case int16:
		return strconv.FormatInt(int64(v), 10), true
	case int32:
		return strconv.FormatInt(int64(v), 10), true
	case int64:
		return strconv.FormatInt(v, 10), true
	case uint:
		return strconv.FormatUint(uint64(v), 10), true
	case uint8:
		return strconv.FormatUint(uint64(v), 10), true
	case uint16:
		return strconv.FormatUint(uint64(v), 10), true
	case uint32:
		return strconv.FormatUint(uint64(v), 10), true
	case uint64:
		return strconv.FormatUint(v, 10), true
	case float32:
		return strconv.FormatFloat(float64(v), 'g', -1, 32), true
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64), true
	case json.Number:
		return v.String(), true
	case []int:
		data, err := json.Marshal(v)
		if err != nil {
			return "", false
		}
		return string(data), true
	case []byte:
		data, err := json.Marshal(v)
		if err != nil {
			return "", false
		}
		return string(data), true
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return "", false
		}
		return string(data), true
	}
}

func mergePropertyMessages(messages []propertyMessage) ([]byte, error) {
	merged := propertyMessage{Type: "property"}
	for _, message := range messages {
		if message.Type != "" {
			merged.Type = message.Type
		}
		merged.Payload = append(merged.Payload, message.Payload...)
	}
	return json.Marshal(merged)
}
