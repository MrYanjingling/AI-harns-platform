package collectionmgr

import (
	"context"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/validation"
)

// SubscribeFrames returns a live frame channel for one collection.
func (s *Service) SubscribeFrames(ctx context.Context, id string, requests bool) (<-chan collection.FrameEvent, func(), error) {
	record, err := s.getCollection(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if !record.Enable {
		return nil, nil, validation.New([]validation.FieldIssue{{Field: "enable", MsgID: "validate.collection_disabled"}})
	}
	if s.runtime == nil {
		return nil, nil, validation.New([]validation.FieldIssue{{
			Field: "collection", MsgID: "validate.collection_not_running",
		}})
	}
	hub, ok := s.runtime.Trace(id)
	if !ok {
		return nil, nil, validation.New([]validation.FieldIssue{{
			Field: "collection", MsgID: "validate.collection_not_running",
		}})
	}
	ch, cancel := hub.Subscribe(requests)
	return ch, cancel, nil
}
