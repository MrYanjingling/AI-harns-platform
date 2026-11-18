package collectionmgr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/validation"
	"harns-platform/pkg/logging"
)

const (
	defaultRetryInterval    = 15 * time.Second
	defaultRetryMaxInterval = 5 * time.Minute
	defaultRetryTimeout     = 5 * time.Second
	defaultRetryMultiplier  = 2
	defaultRetryJitter      = 0.2
	retryScanInterval       = time.Second
)

// ErrFactoryNotFound reports that no collector factory matches a driver.
var ErrFactoryNotFound = errors.New("collector factory not found")

type retryPolicy struct {
	initial    time.Duration
	max        time.Duration
	timeout    time.Duration
	multiplier float64
	jitter     float64
}

func defaultRetryPolicy() retryPolicy {
	return retryPolicy{
		initial:    defaultRetryInterval,
		max:        defaultRetryMaxInterval,
		timeout:    defaultRetryTimeout,
		multiplier: defaultRetryMultiplier,
		jitter:     defaultRetryJitter,
	}
}

func (p retryPolicy) delay(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	initial := p.initial
	if initial <= 0 {
		initial = defaultRetryInterval
	}
	maxInterval := p.max
	if maxInterval < initial {
		maxInterval = initial
	}
	multiplier := p.multiplier
	if multiplier < 1 {
		multiplier = 1
	}
	delay := float64(initial) * math.Pow(multiplier, float64(attempts-1))
	if delay > float64(maxInterval) {
		delay = float64(maxInterval)
	}
	if p.jitter > 0 {
		delay *= 1 + (rand.Float64()*2-1)*p.jitter
	}
	if delay < float64(initial) {
		delay = float64(initial)
	}
	if delay > float64(maxInterval) {
		delay = float64(maxInterval)
	}
	return time.Duration(delay)
}

type retryState struct {
	mu          sync.Mutex
	attempts    int
	nextRetryAt time.Time
	retrying    bool
}

// RuntimeCollection is one collection loaded in memory.
type RuntimeCollection struct {
	Collector collection.Collector
	Trace     *collection.TraceHub

	opMu      sync.Mutex
	errMu     sync.RWMutex
	lastError string
}

func (r *RuntimeCollection) SetLastError(err error) {
	if r == nil {
		return
	}
	r.errMu.Lock()
	defer r.errMu.Unlock()
	if err == nil {
		r.lastError = ""
		return
	}
	r.lastError = err.Error()
}

func (r *RuntimeCollection) ClearLastError() {
	if r == nil {
		return
	}
	r.errMu.Lock()
	defer r.errMu.Unlock()
	r.lastError = ""
}

func (r *RuntimeCollection) LastError() string {
	if r == nil {
		return ""
	}
	r.errMu.RLock()
	defer r.errMu.RUnlock()
	return r.lastError
}

// RuntimeManager owns the in-memory collection runtime.
type RuntimeManager struct {
	collections sync.Map // collectionID -> *RuntimeCollection
	hubs        sync.Map // collectionID -> *collection.TraceHub
	retryStates sync.Map // collectionID -> *retryState
	factories   map[driverKey]collection.CollectorFactory
	retryPolicy retryPolicy

	stopCh    chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
}

// NewRuntimeManager creates a runtime manager and registers factories.
func NewRuntimeManager(factories ...collection.CollectorFactory) (*RuntimeManager, error) {
	manager := &RuntimeManager{
		factories:   make(map[driverKey]collection.CollectorFactory, len(factories)),
		retryPolicy: defaultRetryPolicy(),
		stopCh:      make(chan struct{}),
	}
	for _, factory := range factories {
		if factory == nil {
			return nil, fmt.Errorf("collector factory is nil")
		}
		key := driverKey{driverType: factory.DriverType(), driverModel: factory.DriverModel()}
		if _, exists := manager.factories[key]; exists {
			return nil, fmt.Errorf("duplicate collector factory %s/%s", key.driverType, key.driverModel)
		}
		manager.factories[key] = factory
	}
	return manager, nil
}

// Start starts the unconnected retry loop.
func (m *RuntimeManager) Start() {
	if m == nil {
		return
	}
	m.wg.Add(1)
	go m.retryLoop()
}

// Load loads persisted collections into memory.
func (m *RuntimeManager) Load(ctx context.Context, records []*CollectionRecord) error {
	var errs []error
	for _, record := range records {
		if err := m.Register(ctx, record); err != nil {
			errs = append(errs, fmt.Errorf("register collection %s: %w", record.ID, err))
		}
	}
	return errors.Join(errs...)
}

// Register creates and stores one collector, then connects and starts it.
func (m *RuntimeManager) Register(ctx context.Context, record *CollectionRecord) error {
	if m == nil || record == nil {
		return nil
	}
	m.deleteRetryState(record.ID)
	runtime, err := m.newRuntime(record)
	if err != nil {
		return err
	}
	m.collections.Store(record.ID, runtime)
	m.connectAndStart(ctx, runtime)
	return nil
}

// Replace stops the old collector and installs a new one.
func (m *RuntimeManager) Replace(ctx context.Context, record *CollectionRecord) error {
	if m == nil || record == nil {
		return nil
	}
	m.deleteRetryState(record.ID)
	if old, ok := m.collections.LoadAndDelete(record.ID); ok {
		stopRuntime(ctx, old.(*RuntimeCollection))
	}
	return m.Register(ctx, record)
}

// Remove stops and removes one collection.
func (m *RuntimeManager) Remove(ctx context.Context, id string) error {
	if m == nil {
		return nil
	}
	m.deleteRetryState(id)
	if current, ok := m.collections.LoadAndDelete(id); ok {
		stopRuntime(ctx, current.(*RuntimeCollection))
	}
	if hub, ok := m.hubs.LoadAndDelete(id); ok {
		hub.(*collection.TraceHub).Close()
	}
	return nil
}

// Restart stops, reconnects, and restarts one collection.
func (m *RuntimeManager) Restart(ctx context.Context, id string) error {
	current, ok := m.Get(id)
	if !ok {
		return ErrCollectionNotFound
	}
	m.deleteRetryState(id)
	current.opMu.Lock()
	defer current.opMu.Unlock()
	defer m.refreshRetryState(id, current)
	_ = current.Collector.Stop(ctx)
	if err := current.Collector.Connect(ctx); err != nil {
		current.SetLastError(err)
		if errors.Is(err, collection.ErrNotImplemented) || errors.Is(err, collection.ErrInvalidConfig) {
			current.Collector.SetCollectStatus(collection.CollectStatusError)
		} else {
			current.Collector.SetCollectStatus(collection.CollectStatusUnconnected)
		}
		return err
	}
	if err := current.Collector.Start(ctx); err != nil {
		current.SetLastError(err)
		current.Collector.SetCollectStatus(collection.CollectStatusError)
		return err
	}
	current.ClearLastError()
	return nil
}

// DeliverAction serializes a write action against this runtime lifecycle.
func (r *RuntimeCollection) DeliverAction(ctx context.Context, actions []collection.ActionValue) error {
	if r == nil || r.Collector == nil {
		return fmt.Errorf("collection runtime is nil")
	}
	handler, ok := r.Collector.(collection.ActionHandler)
	if !ok {
		return fmt.Errorf("collection action is not supported")
	}
	r.opMu.Lock()
	defer r.opMu.Unlock()
	switch r.Collector.GetCollectStatus() {
	case collection.CollectStatusStopped, collection.CollectStatusUnconnected,
		collection.CollectStatusEmptyVariable, collection.CollectStatusError:
		return validation.New([]validation.FieldIssue{{Field: "collection", MsgID: "validate.action_collection_not_ready"}})
	}
	return handler.DeliverAction(ctx, actions)
}

// StopCollection pauses one collection while keeping its runtime and trace hub.
func (m *RuntimeManager) StopCollection(ctx context.Context, id, reason string) error {
	current, ok := m.Get(id)
	if !ok {
		return ErrCollectionNotFound
	}
	current.opMu.Lock()
	defer current.opMu.Unlock()
	if err := current.Collector.Stop(ctx); err != nil {
		current.SetLastError(err)
		return err
	}
	current.Collector.SetCollectStatus(collection.CollectStatusStopped)
	current.ClearLastError()
	m.deleteRetryState(id)
	return nil
}

// StopAll pauses every loaded collection while keeping records and trace hubs.
func (m *RuntimeManager) StopAll(ctx context.Context, reason string) error {
	if m == nil {
		return nil
	}
	var errs []error
	m.collections.Range(func(key, _ interface{}) bool {
		id, ok := key.(string)
		if !ok {
			return true
		}
		if err := m.StopCollection(ctx, id, reason); err != nil && !errors.Is(err, ErrCollectionNotFound) {
			errs = append(errs, err)
		}
		return true
	})
	return errors.Join(errs...)
}

// Get returns one runtime collection.
func (m *RuntimeManager) Get(id string) (*RuntimeCollection, bool) {
	if m == nil {
		return nil, false
	}
	current, ok := m.collections.Load(id)
	if !ok {
		return nil, false
	}
	runtime, ok := current.(*RuntimeCollection)
	return runtime, ok
}

// SnapshotVariableValues returns runtime values for requested variables.
func (m *RuntimeManager) SnapshotVariableValues(id string, names []string) map[string]any {
	runtime, ok := m.Get(id)
	if !ok {
		return nil
	}
	return runtime.Collector.GetCollectionMeta().SnapshotVariableValues(names)
}

// SnapshotVariables returns runtime variable values for one collection.
func (m *RuntimeManager) SnapshotVariables(id string) []*collection.Variable {
	runtime, ok := m.Get(id)
	if !ok {
		return nil
	}
	return runtime.Collector.GetCollectionMeta().SnapshotVariables()
}

// IDs returns the currently loaded collection IDs.
func (m *RuntimeManager) IDs() []string {
	if m == nil {
		return nil
	}
	ids := make([]string, 0)
	m.collections.Range(func(key, _ interface{}) bool {
		id, ok := key.(string)
		if ok {
			ids = append(ids, id)
		}
		return true
	})
	sort.Strings(ids)
	return ids
}

// Status returns the live collection status.
func (m *RuntimeManager) Status(id string) (string, bool) {
	runtime, ok := m.Get(id)
	if !ok {
		return "", false
	}
	return runtime.Collector.GetCollectStatus(), true
}

// LastError returns the latest runtime error for one collection.
func (m *RuntimeManager) LastError(id string) (string, bool) {
	runtime, ok := m.Get(id)
	if !ok {
		return "", false
	}
	return runtime.LastError(), true
}

// Close stops all collectors and the retry loop.
func (m *RuntimeManager) Close(ctx context.Context) error {
	if m == nil {
		return nil
	}
	m.closeOnce.Do(func() { close(m.stopCh) })
	m.wg.Wait()
	m.collections.Range(func(key, value interface{}) bool {
		stopRuntime(ctx, value.(*RuntimeCollection))
		m.collections.Delete(key)
		return true
	})
	m.retryStates.Range(func(key, _ interface{}) bool {
		m.retryStates.Delete(key)
		return true
	})
	m.hubs.Range(func(key, value interface{}) bool {
		value.(*collection.TraceHub).Close()
		m.hubs.Delete(key)
		return true
	})
	return nil
}

func (m *RuntimeManager) newRuntime(record *CollectionRecord) (*RuntimeCollection, error) {
	factory, ok := m.factories[driverKey{driverType: record.DriverType, driverModel: record.DriverModel}]
	if !ok {
		return nil, fmt.Errorf("%w: %s/%s", ErrFactoryNotFound, record.DriverType, record.DriverModel)
	}
	collector, err := factory.NewCollector(&collection.CollectionMeta{
		Meta: collection.Meta{
			Name:    record.Name,
			ID:      record.ID,
			Version: record.Version,
			ModTime: record.UpdatedTime,
		},
		DriverType:       record.DriverType,
		DriverModel:      record.DriverModel,
		CollectStatus:    record.CollectStatus,
		CollectorCycleMs: record.CollectorCycleMs,
		Variables:        record.Variables,
	}, json.RawMessage(record.ProtocolConfig))
	if err != nil {
		return nil, err
	}
	hub := m.traceHub(record.ID)
	if aware, ok := collector.(collection.FrameTracerAware); ok {
		aware.SetFrameTracer(hub)
	}
	runtime := &RuntimeCollection{Collector: collector, Trace: hub}
	collector.GetCollectionMeta().SetErrorReporter(runtime.SetLastError)
	return runtime, nil
}

func (m *RuntimeManager) traceHub(id string) *collection.TraceHub {
	if hub, ok := m.hubs.Load(id); ok {
		return hub.(*collection.TraceHub)
	}
	hub := collection.NewTraceHub()
	actual, _ := m.hubs.LoadOrStore(id, hub)
	return actual.(*collection.TraceHub)
}

// Trace returns the live trace hub for one collection.
func (m *RuntimeManager) Trace(id string) (*collection.TraceHub, bool) {
	if m == nil {
		return nil, false
	}
	runtime, ok := m.Get(id)
	if !ok || runtime.Trace == nil {
		return nil, false
	}
	return runtime.Trace, true
}

func (m *RuntimeManager) connectAndStart(ctx context.Context, runtime *RuntimeCollection) {
	runtime.opMu.Lock()
	defer runtime.opMu.Unlock()
	id := runtime.Collector.GetCollectionMeta().GetID()
	defer m.refreshRetryState(id, runtime)
	m.connectAndStartLocked(ctx, runtime)
}

func (m *RuntimeManager) connectAndStartLocked(ctx context.Context, runtime *RuntimeCollection) {
	if err := runtime.Collector.Connect(ctx); err != nil {
		runtime.SetLastError(err)
		if errors.Is(err, collection.ErrNotImplemented) || errors.Is(err, collection.ErrInvalidConfig) {
			runtime.Collector.SetCollectStatus(collection.CollectStatusError)
		} else {
			runtime.Collector.SetCollectStatus(collection.CollectStatusUnconnected)
		}
		return
	}
	if err := runtime.Collector.Start(ctx); err != nil {
		runtime.SetLastError(err)
		runtime.Collector.SetCollectStatus(collection.CollectStatusError)
		return
	}
	runtime.ClearLastError()
}

func (m *RuntimeManager) retryLoop() {
	defer m.wg.Done()
	ticker := time.NewTicker(retryScanInterval)
	defer ticker.Stop()
	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.retryUnconnected()
		}
	}
}

func (m *RuntimeManager) retryUnconnected() {
	now := time.Now()
	m.collections.Range(func(key, value interface{}) bool {
		id, ok := key.(string)
		if !ok {
			return true
		}
		runtime := value.(*RuntimeCollection)
		if runtime.Collector.GetCollectStatus() != collection.CollectStatusUnconnected {
			m.retryStates.Delete(id)
			return true
		}
		state := m.ensureRetryState(id)
		state.mu.Lock()
		if state.retrying || now.Before(state.nextRetryAt) {
			state.mu.Unlock()
			return true
		}
		state.retrying = true
		state.mu.Unlock()
		m.wg.Add(1)
		go m.retryOne(id, runtime, state)
		return true
	})
}

func (m *RuntimeManager) retryOne(id string, runtime *RuntimeCollection, state *retryState) {
	defer m.wg.Done()
	ctx, cancel := context.WithTimeout(context.Background(), m.retryPolicy.timeout)
	defer cancel()

	runtime.opMu.Lock()
	connectErr := runtime.Collector.Connect(ctx)
	var startErr error
	if connectErr == nil {
		startErr = runtime.Collector.Start(ctx)
	}
	runtime.opMu.Unlock()

	now := time.Now()
	state.mu.Lock()
	defer state.mu.Unlock()
	state.retrying = false

	switch {
	case connectErr != nil:
		runtime.SetLastError(connectErr)
		if errors.Is(connectErr, collection.ErrInvalidConfig) || errors.Is(connectErr, collection.ErrNotImplemented) {
			runtime.Collector.SetCollectStatus(collection.CollectStatusError)
			m.retryStates.Delete(id)
			return
		}
		runtime.Collector.SetCollectStatus(collection.CollectStatusUnconnected)
		state.attempts++
		state.nextRetryAt = now.Add(m.retryPolicy.delay(state.attempts))
	case startErr != nil:
		runtime.SetLastError(startErr)
		runtime.Collector.SetCollectStatus(collection.CollectStatusError)
		m.retryStates.Delete(id)
	default:
		runtime.ClearLastError()
		m.retryStates.Delete(id)
	}
}

func (m *RuntimeManager) ensureRetryState(id string) *retryState {
	state := &retryState{}
	actual, _ := m.retryStates.LoadOrStore(id, state)
	return actual.(*retryState)
}

func (m *RuntimeManager) refreshRetryState(id string, runtime *RuntimeCollection) {
	if runtime == nil || runtime.Collector.GetCollectStatus() != collection.CollectStatusUnconnected {
		m.retryStates.Delete(id)
		return
	}
	state := m.ensureRetryState(id)
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.attempts == 0 {
		state.attempts = 1
	}
	if state.nextRetryAt.IsZero() {
		state.nextRetryAt = time.Now().Add(m.retryPolicy.delay(state.attempts))
	}
}

func (m *RuntimeManager) deleteRetryState(id string) {
	if m == nil {
		return
	}
	m.retryStates.Delete(id)
}

func stopRuntime(ctx context.Context, runtime *RuntimeCollection) {
	if runtime == nil || runtime.Collector == nil {
		return
	}
	runtime.opMu.Lock()
	defer runtime.opMu.Unlock()
	if err := runtime.Collector.Stop(ctx); err != nil {
		logging.ErrorS(err, "Failed to stop collection", "collectionId", runtime.Collector.GetCollectionMeta().GetID())
	}
	runtime.ClearLastError()
}
