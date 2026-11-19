package collectionmgr

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/collection"
)

type fakeCollector struct {
	meta       *collection.CollectionMeta
	connectErr error
	startErr   error
	mu         sync.Mutex
	connects   int
	starts     int
	stops      int
}

func (c *fakeCollector) GetCollectionMeta() *collection.CollectionMeta { return c.meta }
func (c *fakeCollector) SetCollectionMeta(meta *collection.CollectionMeta) {
	c.meta = meta
}
func (c *fakeCollector) Connect(context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connects++
	if c.connectErr != nil {
		c.meta.SetCollectStatus(collection.CollectStatusUnconnected)
		return c.connectErr
	}
	c.meta.SetCollectStatus(collection.CollectStatusStopped)
	return nil
}
func (c *fakeCollector) Start(context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.starts++
	if c.startErr != nil {
		c.meta.SetCollectStatus(collection.CollectStatusError)
		return c.startErr
	}
	c.meta.SetCollectStatus(collection.CollectStatusCollecting)
	return nil
}
func (c *fakeCollector) Stop(context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stops++
	c.meta.SetCollectStatus(collection.CollectStatusStopped)
	return nil
}
func (c *fakeCollector) GetCollectStatus() string { return c.meta.GetCollectStatus() }
func (c *fakeCollector) SetCollectStatus(status string) {
	c.meta.SetCollectStatus(status)
}
func (c *fakeCollector) IndexCollection() { c.meta.IndexCollection() }

type fakeCollectorFactory struct {
	mu         sync.Mutex
	collectors []*fakeCollector
	connectErr error
	startErr   error
}

func (f *fakeCollectorFactory) DriverType() string  { return "test" }
func (f *fakeCollectorFactory) DriverModel() string { return "model" }
func (f *fakeCollectorFactory) NewCollector(meta *collection.CollectionMeta, _ json.RawMessage) (collection.Collector, error) {
	meta.IndexCollection()
	c := &fakeCollector{meta: meta, connectErr: f.connectErr, startErr: f.startErr}
	f.mu.Lock()
	f.collectors = append(f.collectors, c)
	f.mu.Unlock()
	return c, nil
}

func fakeRecord(id string) *CollectionRecord {
	return &CollectionRecord{
		ID:               id,
		Name:             "collection-1",
		Version:          "1",
		DriverType:       "test",
		DriverModel:      "model",
		CollectStatus:    collection.CollectStatusStopped,
		CollectorCycleMs: 1000,
		ProtocolConfig:   json.RawMessage(`{}`),
		Variables: []*collection.Variable{{
			Name:       "temperature",
			DataType:   "float32",
			Address:    "1!40001",
			AccessMode: collection.AccessModeRead,
		}},
	}
}

func TestRuntimeManagerRegisterReplaceRemove(t *testing.T) {
	factory := &fakeCollectorFactory{}
	manager, err := NewRuntimeManager(factory)
	require.NoError(t, err)

	record := fakeRecord("c1")
	require.NoError(t, manager.Register(context.Background(), record))
	runtime, ok := manager.Get("c1")
	require.True(t, ok)
	assert.Equal(t, collection.CollectStatusCollecting, runtime.Collector.GetCollectStatus())
	assert.Len(t, factory.collectors, 1)
	assert.Equal(t, 1, factory.collectors[0].connects)
	assert.Equal(t, 1, factory.collectors[0].starts)

	factory.collectors[0].meta.SetVariableValue("temperature", float64(26.5))
	variables := manager.SnapshotVariables("c1")
	require.Len(t, variables, 1)
	assert.Equal(t, float64(26.5), variables[0].Value)

	updated := fakeRecord("c1")
	updated.Name = "collection-2"
	require.NoError(t, manager.Replace(context.Background(), updated))
	assert.Equal(t, 1, factory.collectors[0].stops)
	assert.Len(t, factory.collectors, 2)
	assert.Equal(t, "collection-2", factory.collectors[1].meta.Name)

	require.NoError(t, manager.Remove(context.Background(), "c1"))
	assert.Equal(t, 1, factory.collectors[1].stops)
	_, ok = manager.Get("c1")
	assert.False(t, ok)
}

func TestRuntimeManagerNotImplementedStatus(t *testing.T) {
	factory := &fakeCollectorFactory{connectErr: collection.ErrNotImplemented}
	manager, err := NewRuntimeManager(factory)
	require.NoError(t, err)

	require.NoError(t, manager.Register(context.Background(), fakeRecord("c1")))
	runtime, ok := manager.Get("c1")
	require.True(t, ok)
	assert.Equal(t, collection.CollectStatusError, runtime.Collector.GetCollectStatus())
	assert.Equal(t, collection.ErrNotImplemented.Error(), runtime.LastError())
	assert.Equal(t, 0, factory.collectors[0].starts)
}

func TestRuntimeManagerClose(t *testing.T) {
	factory := &fakeCollectorFactory{}
	manager, err := NewRuntimeManager(factory)
	require.NoError(t, err)
	manager.Start()
	require.NoError(t, manager.Register(context.Background(), fakeRecord("c1")))
	require.NoError(t, manager.Close(context.Background()))
	assert.Equal(t, 1, factory.collectors[0].stops)
}

func TestRuntimeManagerSnapshotVariableValues(t *testing.T) {
	factory := &fakeCollectorFactory{}
	manager, err := NewRuntimeManager(factory)
	require.NoError(t, err)
	require.NoError(t, manager.Register(context.Background(), fakeRecord("c1")))

	factory.collectors[0].meta.SetVariableValue("temperature", float64(26.5))
	values := manager.SnapshotVariableValues("c1", []string{"temperature", "missing"})
	require.Len(t, values, 1)
	assert.Equal(t, float64(26.5), values["temperature"])
	assert.Nil(t, manager.SnapshotVariableValues("missing", []string{"temperature"}))
}

func TestRuntimeManagerStopCollectionKeepsRuntime(t *testing.T) {
	factory := &fakeCollectorFactory{}
	manager, err := NewRuntimeManager(factory)
	require.NoError(t, err)
	require.NoError(t, manager.Register(context.Background(), fakeRecord("c1")))

	factory.collectors[0].meta.SetVariableValue("temperature", float64(26.5))
	hubBefore, ok := manager.Trace("c1")
	require.True(t, ok)
	require.NoError(t, manager.StopCollection(context.Background(), "c1", "manual"))
	runtime, ok := manager.Get("c1")
	require.True(t, ok)
	assert.Equal(t, collection.CollectStatusStopped, runtime.Collector.GetCollectStatus())
	assert.Equal(t, 1, factory.collectors[0].stops)
	values := manager.SnapshotVariableValues("c1", []string{"temperature"})
	value, ok := values["temperature"]
	assert.True(t, ok)
	assert.Nil(t, value)
	hubAfter, ok := manager.Trace("c1")
	require.True(t, ok)
	assert.Same(t, hubBefore, hubAfter)
}

type fakeActionCollector struct {
	fakeCollector
	delivered int
}

func (c *fakeActionCollector) DeliverAction(context.Context, []collection.ActionValue) error {
	c.delivered++
	return nil
}

func TestRuntimeCollectionDeliverActionAllowsCollectingStates(t *testing.T) {
	meta := &collection.CollectionMeta{CollectStatus: collection.CollectStatusCollectingError}
	collector := &fakeActionCollector{fakeCollector: fakeCollector{meta: meta}}
	runtime := &RuntimeCollection{Collector: collector}

	require.NoError(t, runtime.DeliverAction(context.Background(), []collection.ActionValue{{}}))
	assert.Equal(t, 1, collector.delivered)

	meta.SetCollectStatus(collection.CollectStatusCollecting)
	require.NoError(t, runtime.DeliverAction(context.Background(), []collection.ActionValue{{}}))
	assert.Equal(t, 2, collector.delivered)

	meta.SetCollectStatus(collection.CollectStatusStopped)
	require.Error(t, runtime.DeliverAction(context.Background(), []collection.ActionValue{{}}))
	assert.Equal(t, 2, collector.delivered)
}

func TestRetryPolicyDelay(t *testing.T) {
	policy := retryPolicy{
		initial:    15 * time.Second,
		max:        5 * time.Minute,
		multiplier: 2,
	}
	assert.Equal(t, 15*time.Second, policy.delay(1))
	assert.Equal(t, 30*time.Second, policy.delay(2))
	assert.Equal(t, time.Minute, policy.delay(3))
	assert.Equal(t, 2*time.Minute, policy.delay(4))
	assert.Equal(t, 4*time.Minute, policy.delay(5))
	assert.Equal(t, 5*time.Minute, policy.delay(6))
	assert.Equal(t, 5*time.Minute, policy.delay(20))
}

func TestRuntimeManagerRetryBackoffAndRecovery(t *testing.T) {
	connectErr := errors.New("dial failed")
	factory := &fakeCollectorFactory{connectErr: connectErr}
	manager, err := NewRuntimeManager(factory)
	require.NoError(t, err)
	manager.retryPolicy = retryPolicy{
		initial:    10 * time.Millisecond,
		max:        80 * time.Millisecond,
		timeout:    50 * time.Millisecond,
		multiplier: 2,
	}

	require.NoError(t, manager.Register(context.Background(), fakeRecord("c1")))
	runtime, ok := manager.Get("c1")
	require.True(t, ok)
	assert.Equal(t, collection.CollectStatusUnconnected, runtime.Collector.GetCollectStatus())
	assert.Equal(t, connectErr.Error(), runtime.LastError())

	state := manager.ensureRetryState("c1")
	manager.wg.Add(1)
	manager.retryOne("c1", runtime, state)
	state.mu.Lock()
	assert.Equal(t, 2, state.attempts)
	assert.False(t, state.retrying)
	assert.True(t, state.nextRetryAt.After(time.Now()))
	state.mu.Unlock()

	factory.collectors[0].mu.Lock()
	factory.collectors[0].connectErr = nil
	factory.collectors[0].mu.Unlock()
	manager.wg.Add(1)
	manager.retryOne("c1", runtime, state)
	assert.Equal(t, collection.CollectStatusCollecting, runtime.Collector.GetCollectStatus())
	assert.Empty(t, runtime.LastError())
	_, exists := manager.retryStates.Load("c1")
	assert.False(t, exists)
}

func TestRuntimeCollectionLastErrorFromReporter(t *testing.T) {
	runtime := &RuntimeCollection{}
	meta := &collection.CollectionMeta{}
	meta.SetErrorReporter(runtime.SetLastError)

	meta.ReportError(errors.New("read failed"))
	assert.Equal(t, "read failed", runtime.LastError())
	meta.ReportError(nil)
	assert.Empty(t, runtime.LastError())
}

func TestRuntimeManagerStopAll(t *testing.T) {
	factory := &fakeCollectorFactory{}
	manager, err := NewRuntimeManager(factory)
	require.NoError(t, err)
	require.NoError(t, manager.Register(context.Background(), fakeRecord("c1")))
	require.NoError(t, manager.Register(context.Background(), fakeRecord("c2")))

	require.NoError(t, manager.StopAll(context.Background(), "license deadline"))
	for _, id := range []string{"c1", "c2"} {
		runtime, ok := manager.Get(id)
		require.True(t, ok)
		assert.Equal(t, collection.CollectStatusStopped, runtime.Collector.GetCollectStatus())
	}
	assert.Equal(t, 2, factory.collectors[0].stops+factory.collectors[1].stops)
}

type copiedMetaCollector struct {
	meta *collection.CollectionMeta
}

func (c *copiedMetaCollector) GetCollectionMeta() *collection.CollectionMeta { return c.meta }
func (c *copiedMetaCollector) SetCollectionMeta(meta *collection.CollectionMeta) {
	c.meta = &collection.CollectionMeta{}
	c.meta.SetCollectionMeta(meta)
}
func (c *copiedMetaCollector) Connect(context.Context) error { return nil }
func (c *copiedMetaCollector) Start(context.Context) error   { return nil }
func (c *copiedMetaCollector) Stop(context.Context) error    { return nil }
func (c *copiedMetaCollector) GetCollectStatus() string      { return c.meta.GetCollectStatus() }
func (c *copiedMetaCollector) SetCollectStatus(status string) {
	c.meta.SetCollectStatus(status)
}
func (c *copiedMetaCollector) IndexCollection() { c.meta.IndexCollection() }

type copiedMetaFactory struct{}

func (f *copiedMetaFactory) DriverType() string  { return "copy-test" }
func (f *copiedMetaFactory) DriverModel() string { return "copy-test" }
func (f *copiedMetaFactory) NewCollector(
	meta *collection.CollectionMeta,
	_ json.RawMessage,
) (collection.Collector, error) {
	collector := &copiedMetaCollector{}
	collector.SetCollectionMeta(meta)
	return collector, nil
}

func TestRuntimeReportsCollectorMetaErrors(t *testing.T) {
	manager, err := NewRuntimeManager(&copiedMetaFactory{})
	require.NoError(t, err)
	record := fakeRecord("c1")
	record.DriverType = "copy-test"
	record.DriverModel = "copy-test"
	runtime, err := manager.newRuntime(record)
	require.NoError(t, err)

	runtime.Collector.GetCollectionMeta().ReportError(errors.New("calculation expression failed"))
	assert.Equal(t, "calculation expression failed", runtime.LastError())

	runtime.Collector.GetCollectionMeta().ReportError(nil)
	assert.Empty(t, runtime.LastError())
}

func TestRetryUnconnectedKeepsStateWhileRetrying(t *testing.T) {
	factory := &fakeCollectorFactory{}
	manager, err := NewRuntimeManager(factory)
	require.NoError(t, err)
	require.NoError(t, manager.Register(context.Background(), fakeRecord("c1")))
	runtime, ok := manager.Get("c1")
	require.True(t, ok)
	runtime.Collector.SetCollectStatus(collection.CollectStatusStopped)

	state := manager.ensureRetryState("c1")
	state.mu.Lock()
	state.retrying = true
	state.mu.Unlock()

	manager.retryUnconnected()

	_, exists := manager.retryStates.Load("c1")
	assert.True(t, exists)
}
