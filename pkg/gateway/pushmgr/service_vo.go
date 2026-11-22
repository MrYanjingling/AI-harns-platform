package pushmgr

import (
	"time"

	"harns-platform/pkg/gateway/pushmgr/v1"
)

func (s *Service) connectorVO(item *ConnectorRecord) *v1.ConnectorVo {
	if item == nil {
		return nil
	}
	status := ConnectorStatusStopped
	lastError := ""
	if s != nil && s.runtime != nil {
		status = s.runtime.ConnectorStatus(item.ID)
		lastError = s.runtime.ConnectorLastError(item.ID)
	}
	return &v1.ConnectorVo{
		ID:          item.ID,
		Type:        item.Type,
		Name:        item.Name,
		PushConfig:  item.PushConfig,
		RunState:    item.RunState,
		GroupCount:  item.GroupCount,
		Status:      status,
		LastError:   lastError,
		Version:     item.Version,
		CreatedTime: item.CreatedTime,
		UpdatedTime: item.UpdatedTime,
	}
}

func (s *Service) groupVO(item *PushGroup) *v1.PushGroupVo {
	if item == nil {
		return nil
	}
	status := ConnectorStatusStopped
	lastError := ""
	pushError := ""
	subscriptionError := ""
	var (
		lastPushTime      *time.Time
		pendingMessages   int64
		pendingBatches    int64
		oldestPendingTime *time.Time
	)
	if s != nil && s.runtime != nil {
		status = s.runtime.GroupStatus(item.ID)
		lastError = s.runtime.GroupLastError(item.ID)
		pushError, subscriptionError = s.runtime.GroupErrors(item.ID)
		if value := s.runtime.GroupLastPushTime(item.ID); !value.IsZero() {
			lastPushTime = &value
		}
		var oldest time.Time
		pendingMessages, pendingBatches, oldest = s.runtime.GroupPending(item.ID)
		if !oldest.IsZero() {
			oldestPendingTime = &oldest
		}
	}
	return &v1.PushGroupVo{
		ID:                           item.ID,
		Name:                         item.Name,
		ConnectorID:                  item.ConnectorID,
		DataTopic:                    item.DataTopic,
		CommandTopic:                 item.CommandTopic,
		RunState:                     item.RunState,
		Status:                       status,
		LastError:                    lastError,
		PushError:                    pushError,
		SubscriptionError:            subscriptionError,
		LastPushTime:                 lastPushTime,
		PendingMessages:              pendingMessages,
		PendingBatches:               pendingBatches,
		OldestPendingTime:            oldestPendingTime,
		PushIntervalMs:               item.PushIntervalMs,
		AllCollections:               item.AllCollections,
		CollectionIDs:                item.CollectionIDs,
		MaxCollectionsPerPushMessage: item.MaxCollectionsPerPushMessage,
		Version:                      item.Version,
		CreatedTime:                  item.CreatedTime,
		UpdatedTime:                  item.UpdatedTime,
	}
}
