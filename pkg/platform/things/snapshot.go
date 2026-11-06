package things

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/logging"
)

const (
	snapshotCachePrefix = "iot:thing:snapshot:"
	snapshotCacheTTL    = 24 * time.Hour
)

// ThingSnapshotResolver resolves a cached thing snapshot.
type ThingSnapshotResolver interface {
	GetThingSnapshot(ctx context.Context, thingID string) (*ThingSnapshot, error)
}

// GetThingSnapshot returns one thing snapshot.
func (s *Service) GetThingSnapshot(ctx context.Context, thingID string) (*ThingSnapshot, error) {
	snapshots, err := s.GetThingSnapshots(ctx, []string{thingID})
	if err != nil {
		return nil, err
	}
	snapshot, ok := snapshots[thingID]
	if !ok {
		return nil, response.ErrThingNotFound(thingID)
	}
	return snapshot, nil
}

// GetThingSnapshots returns thing snapshots, using Redis as a shared cache.
func (s *Service) GetThingSnapshots(ctx context.Context, thingIDs []string) (map[string]*ThingSnapshot, error) {
	ids := dedupeIDs(thingIDs)
	result := make(map[string]*ThingSnapshot, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	missing := ids
	if s.redis != nil {
		keys := make([]string, 0, len(ids))
		for _, id := range ids {
			keys = append(keys, snapshotCachePrefix+id)
		}
		values, err := s.redis.MGet(ctx, keys...).Result()
		if err != nil {
			logging.V(2).InfoS("Failed to read thing snapshots", "err", err)
		} else {
			missing = make([]string, 0, len(ids))
			for i, value := range values {
				raw := redisValue(value)
				if raw == "" {
					missing = append(missing, ids[i])
					continue
				}
				var snapshot ThingSnapshot
				if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
					logging.V(2).InfoS("Failed to decode thing snapshot", "thingId", ids[i], "err", err)
					missing = append(missing, ids[i])
					continue
				}
				result[ids[i]] = &snapshot
			}
		}
	}

	if len(missing) > 0 {
		snapshots, err := s.store.GetThingSnapshotsByIDs(ctx, missing)
		if err != nil {
			return nil, err
		}
		for _, snapshot := range snapshots {
			result[snapshot.ID] = snapshot
			if s.redis != nil {
				if data, err := json.Marshal(snapshot); err == nil {
					if err := s.redis.Set(ctx, snapshotCachePrefix+snapshot.ID, data, snapshotCacheTTL).Err(); err != nil {
						logging.V(2).InfoS("Failed to cache thing snapshot", "thingId", snapshot.ID, "err", err)
					}
				}
			}
		}
	}
	return result, nil
}

// GetThingSnapshotByDeviceKey resolves a device key and returns its snapshot.
func (s *Service) GetThingSnapshotByDeviceKey(ctx context.Context, deviceKey string) (*ThingSnapshot, error) {
	thingID, err := s.store.GetThingIDByDeviceKey(ctx, deviceKey)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, response.ErrThingNotFound(deviceKey)
	}
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	return s.GetThingSnapshot(ctx, thingID)
}

func (s *Service) invalidateSnapshotCache(ctx context.Context, thingID string) {
	if s.redis == nil || thingID == "" {
		return
	}
	if err := s.redis.Del(ctx, snapshotCachePrefix+thingID).Err(); err != nil {
		logging.V(2).InfoS("Failed to invalidate thing snapshot", "thingId", thingID, "err", err)
	}
}
