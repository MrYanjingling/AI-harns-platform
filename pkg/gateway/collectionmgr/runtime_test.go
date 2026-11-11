package collectionmgr

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

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

	hubBefore, ok := manager.Trace("c1")
	require.True(t, ok)
	require.NoError(t, manager.StopCollection(context.Background(), "c1", "manual"))
	runtime, ok := manager.Get("c1")
	require.True(t, ok)
	assert.Equal(t, collection.CollectStatusStopped, runtime.Collector.GetCollectStatus())
	assert.Equal(t, 1, factory.collectors[0].stops)
	hubAfter, ok := manager.Trace("c1")
	require.True(t, ok)
	assert.Same(t, hubBefore, hubAfter)
}
