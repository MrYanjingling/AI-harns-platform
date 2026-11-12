package options

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collectionmgr"
	collectionv1 "harns-platform/pkg/gateway/collectionmgr/v1"
	"harns-platform/pkg/gateway/pushmgr"
)

type pushCollectionSource struct {
	runtime *collectionmgr.RuntimeManager
	service *collectionmgr.Service
}

func newPushCollectionSource(runtime *collectionmgr.RuntimeManager, service *collectionmgr.Service) *pushCollectionSource {
	return &pushCollectionSource{runtime: runtime, service: service}
}

func (s *pushCollectionSource) Snapshots(ctx context.Context, all bool, collectionIDs []string) ([]pushmgr.CollectionSnapshot, error) {
	var ids []string
	if all {
		ids = s.runtime.IDs()
	} else {
		ids = append([]string(nil), collectionIDs...)
	}
	sort.Strings(ids)
	now := time.Now().UTC()
	out := make([]pushmgr.CollectionSnapshot, 0, len(ids))
	for _, id := range ids {
		runtime, ok := s.runtime.Get(id)
		if !ok {
			continue
		}
		meta := runtime.Collector.GetCollectionMeta()
		if meta.GetCollectStatus() != collection.CollectStatusCollecting {
			continue
		}
		values := make(map[string]any)
		for _, variable := range meta.SnapshotVariables() {
			if variable != nil {
				values[variable.Name] = variable.Value
			}
		}
		out = append(out, pushmgr.CollectionSnapshot{
			CollectionID: id,
			Status:       meta.GetCollectStatus(),
			Time:         now,
			Values:       values,
		})
	}
	return out, nil
}

func (s *pushCollectionSource) Exists(ctx context.Context, collectionID string) (bool, error) {
	_, err := s.service.Get(ctx, collectionID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *pushCollectionSource) DeliverActions(ctx context.Context, collectionID string, actions []pushmgr.Action) error {
	items := make([]collectionv1.ActionItem, 0, len(actions))
	for _, action := range actions {
		items = append(items, collectionv1.ActionItem{Name: action.Name, Value: action.Value})
	}
	_, err := s.service.DeliverAction(ctx, collectionID, collectionv1.ActionReq{Actions: items})
	return err
}
