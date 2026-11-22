package pushmgr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"harns-platform/pkg/logging"
)

var errPushConnectorUnavailable = errors.New("push connector is not available")

type outboxSink interface {
	publishGroup(ctx context.Context, runtime *groupRuntime, payload []byte) error
	publishStored(ctx context.Context, runtime *groupRuntime, connectorID, topic string, payload []byte) error
	connector(id string) *connectorRuntime
}

type groupOutbox struct {
	runtime *groupRuntime
	store   ReadySendStore
	sink    outboxSink
	memory  []propertyMessage
}

func (m *RuntimeManager) newGroupOutbox(runtime *groupRuntime) *groupOutbox {
	return &groupOutbox{
		runtime: runtime,
		store:   m.readySendStore(),
		sink:    m,
		memory:  make([]propertyMessage, 0, 128),
	}
}

func (o *groupOutbox) run(ctx context.Context, input <-chan []propertyMessage, wake <-chan struct{}, done chan struct{}) {
	defer close(done)
	o.seed(ctx)
	for {
		if o.runtime.backlog.Load() {
			err := o.recover(ctx)
			if err == nil {
				continue
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			if !errors.Is(err, errPushConnectorUnavailable) {
				o.runtime.setPushError(err)
				logging.ErrorS(err, "Failed to publish push payload",
					"groupId", o.runtime.group.ID, "connectorId", o.runtime.group.ConnectorID)
			}
			timer := time.NewTimer(pushInterval(o.runtime))
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			case <-wake:
				timer.Stop()
			case batch := <-input:
				timer.Stop()
				o.appendFailed(batch...)
				if spillErr := o.spill(ctx); spillErr != nil {
					o.runtime.setPushError(spillErr)
				}
			}
			continue
		}

		select {
		case <-ctx.Done():
			return
		case batch := <-input:
			o.sendFresh(ctx, batch)
		case <-wake:
		}
	}
}

func (o *groupOutbox) seed(ctx context.Context) {
	o.runtime.pendingMsgs.Store(int64(len(o.memory)))
	o.runtime.pendingRows.Store(0)
	o.runtime.oldestPending.Store(0)
	o.runtime.backlog.Store(len(o.memory) > 0)
	if o.store == nil {
		return
	}
	stats, err := o.store.ReadySendStats(ctx, o.runtime.group.ID)
	if err != nil {
		o.runtime.setPushError(err)
		logging.ErrorS(err, "Failed to load ready_send stats", "groupId", o.runtime.group.ID)
		return
	}
	o.runtime.pendingMsgs.Store(stats.Messages + int64(len(o.memory)))
	o.runtime.pendingRows.Store(stats.Batches)
	if !stats.Oldest.IsZero() {
		o.runtime.oldestPending.Store(stats.Oldest.UnixNano())
	}
	if stats.Batches > 0 {
		o.runtime.backlog.Store(true)
	}
}

func (o *groupOutbox) sendFresh(ctx context.Context, messages []propertyMessage) {
	if len(messages) == 0 {
		return
	}
	if o.runtime.backlog.Load() {
		o.appendFailed(messages...)
		if err := o.spill(ctx); err != nil {
			o.runtime.setPushError(err)
		}
		return
	}
	for index, message := range messages {
		payload, err := json.Marshal(message)
		if err != nil {
			o.runtime.setPushError(err)
			return
		}
		if err := o.sink.publishGroup(ctx, o.runtime, payload); err != nil {
			if !errors.Is(err, errPushConnectorUnavailable) {
				o.runtime.setPushError(err)
				logging.ErrorS(err, "Failed to publish push payload",
					"groupId", o.runtime.group.ID, "connectorId", o.runtime.group.ConnectorID, "batch", index)
			}
			o.runtime.backlog.Store(true)
			o.appendFailed(messages[index:]...)
			if spillErr := o.spill(ctx); spillErr != nil {
				o.runtime.setPushError(spillErr)
			}
			return
		}
		o.runtime.setPushSuccess(time.Now().UTC())
	}
}

func (o *groupOutbox) recover(ctx context.Context) error {
	if err := o.spill(ctx); err != nil {
		return err
	}

	if len(o.memory) > 0 {
		batch := append([]propertyMessage(nil), o.memory...)
		payload, err := mergePropertyMessages(batch)
		if err != nil {
			return err
		}
		if err := o.sink.publishGroup(ctx, o.runtime, payload); err != nil {
			return err
		}
		o.memory = nil
		o.runtime.pendingMsgs.Add(-int64(len(batch)))
		o.runtime.setPushSuccess(time.Now().UTC())
	}

	for {
		if o.store == nil {
			break
		}
		rows, err := o.store.ListReadySend(ctx, o.runtime.group.ID, 1)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			break
		}
		row := rows[0]
		connector := o.sink.connector(row.ConnectorID)
		if connector == nil {
			_ = o.store.DeleteReadySend(ctx, row.GroupID, row.Seq)
			o.runtime.pendingMsgs.Add(-int64(row.PayloadSize))
			o.runtime.pendingRows.Add(-1)
			continue
		}
		if connector.getStatus() != ConnectorStatusRunning {
			return errPushConnectorUnavailable
		}
		if err := o.sink.publishStored(ctx, o.runtime, row.ConnectorID, row.Topic, row.Payload); err != nil {
			if errors.Is(err, errPushConnectorUnavailable) {
				return err
			}
			_ = o.store.DeleteReadySend(ctx, row.GroupID, row.Seq)
			o.runtime.pendingMsgs.Add(-int64(row.PayloadSize))
			o.runtime.pendingRows.Add(-1)
			o.runtime.setPushError(err)
			logging.ErrorS(err, "Failed to publish push payload",
				"groupId", row.GroupID, "connectorId", row.ConnectorID, "seq", row.Seq)
			continue
		}
		if err := o.store.DeleteReadySend(ctx, row.GroupID, row.Seq); err != nil {
			return err
		}
		o.runtime.pendingMsgs.Add(-int64(row.PayloadSize))
		o.runtime.pendingRows.Add(-1)
		o.runtime.setPushSuccess(time.Now().UTC())
	}

	if len(o.memory) == 0 {
		o.runtime.backlog.Store(false)
		o.runtime.pendingMsgs.Store(0)
		o.runtime.pendingRows.Store(0)
		o.runtime.oldestPending.Store(0)
	}
	return nil
}

func (o *groupOutbox) appendFailed(messages ...propertyMessage) {
	if len(messages) == 0 {
		return
	}
	o.memory = append(o.memory, messages...)
	o.runtime.pendingMsgs.Add(int64(len(messages)))
	if o.runtime.oldestPending.Load() == 0 {
		o.runtime.oldestPending.Store(time.Now().UTC().UnixNano())
	}
}

func (o *groupOutbox) spill(ctx context.Context) error {
	spilled := false
	for len(o.memory) > 100 {
		batch := append([]propertyMessage(nil), o.memory[:100]...)
		payload, err := mergePropertyMessages(batch)
		if err != nil {
			return err
		}
		if o.store == nil {
			return fmt.Errorf("ready_send store is not initialized")
		}
		row := &ReadySendBatch{
			GroupID:     o.runtime.group.ID,
			ConnectorID: o.runtime.group.ConnectorID,
			Topic:       o.runtime.group.DataTopic,
			Payload:     payload,
			PayloadSize: len(batch),
			CreatedTime: time.Now().UTC(),
		}
		if err := o.store.SaveReadySend(ctx, row); err != nil {
			return err
		}
		o.memory = o.memory[100:]
		o.runtime.pendingRows.Add(1)
		spilled = true
	}
	if spilled {
		o.refreshStats(ctx)
	}
	return nil
}

func (o *groupOutbox) flush(ctx context.Context) error {
	if len(o.memory) == 0 {
		return nil
	}
	if o.store == nil {
		return fmt.Errorf("ready_send store is not initialized")
	}
	batch := append([]propertyMessage(nil), o.memory...)
	payload, err := mergePropertyMessages(batch)
	if err != nil {
		return err
	}
	row := &ReadySendBatch{
		GroupID:     o.runtime.group.ID,
		ConnectorID: o.runtime.group.ConnectorID,
		Topic:       o.runtime.group.DataTopic,
		Payload:     payload,
		PayloadSize: len(batch),
		CreatedTime: time.Now().UTC(),
	}
	if err := o.store.SaveReadySend(ctx, row); err != nil {
		return err
	}
	o.memory = nil
	o.runtime.pendingMsgs.Add(-int64(len(batch)))
	o.refreshStats(ctx)
	return nil
}

func (o *groupOutbox) refreshStats(ctx context.Context) {
	if o.store == nil {
		return
	}
	stats, err := o.store.ReadySendStats(ctx, o.runtime.group.ID)
	if err != nil {
		return
	}
	o.runtime.pendingRows.Store(stats.Batches)
	o.runtime.pendingMsgs.Store(stats.Messages + int64(len(o.memory)))
	if !stats.Oldest.IsZero() {
		o.runtime.oldestPending.Store(stats.Oldest.UnixNano())
	} else if len(o.memory) == 0 {
		o.runtime.oldestPending.Store(0)
	}
}

func (m *RuntimeManager) publishGroupPayload(ctx context.Context, runtime *groupRuntime, payload []byte) error {
	connector := m.connectorRuntime(runtime.group.ConnectorID)
	if connector == nil || connector.getStatus() != ConnectorStatusRunning {
		return errPushConnectorUnavailable
	}
	return m.publishEvent(ctx, runtime.hub, connector, runtime.group.ID, runtime.group.ID,
		runtime.group.DataTopic, payload)
}

func (m *RuntimeManager) publishStoredPayload(ctx context.Context, runtime *groupRuntime, connectorID, topic string, payload []byte) error {
	connector := m.connectorRuntime(connectorID)
	if connector == nil || connector.getStatus() != ConnectorStatusRunning {
		return errPushConnectorUnavailable
	}
	return m.publishEvent(ctx, runtime.hub, connector, runtime.group.ID, runtime.group.ID, topic, payload)
}

func (m *RuntimeManager) publishGroup(ctx context.Context, runtime *groupRuntime, payload []byte) error {
	return m.publishGroupPayload(ctx, runtime, payload)
}

func (m *RuntimeManager) publishStored(ctx context.Context, runtime *groupRuntime, connectorID, topic string, payload []byte) error {
	return m.publishStoredPayload(ctx, runtime, connectorID, topic, payload)
}

func (m *RuntimeManager) connector(id string) *connectorRuntime {
	return m.connectorRuntime(id)
}

func (m *RuntimeManager) readySendStore() ReadySendStore {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	store := m.readySend
	m.mu.Unlock()
	return store
}

func (m *RuntimeManager) connectorRuntime(id string) *connectorRuntime {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	connector := m.connectors[id]
	m.mu.Unlock()
	return connector
}

func (m *RuntimeManager) publishEvent(ctx context.Context, hub *payloadHub, connector *connectorRuntime,
	groupID, key, topic string, payload []byte) error {
	started := time.Now()
	err := connector.connector.Publish(ctx, Message{
		Topic:   topic,
		Key:     key,
		Payload: payload,
	})
	event := PushPayloadEvent{
		GroupID:       groupID,
		ConnectorID:   connector.record.ID,
		ConnectorType: connector.record.Type,
		Topic:         topic,
		Key:           key,
		Time:          started.UTC(),
		DurationMs:    time.Since(started).Milliseconds(),
		Payload:       json.RawMessage(payload),
	}
	if err != nil {
		event.Error = err.Error()
	}
	if hub != nil {
		hub.Publish(event)
	}
	return err
}
