// Package opcua implements OPC UA collection.
package opcua

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gopcua/opcua/ua"

	"harns-platform/pkg/gateway/collection"
)

const (
	DriverType  = "opcUa"
	DriverModel = "opcUa"

	DefaultTCPCount = 1
	MaxTCPCount     = 3
)

type ProtocolConfig struct {
	Location string `json:"location"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	TCPCount int    `json:"tcpCount"`
}

func (c *ProtocolConfig) Normalize() {
	if c.Port == 0 {
		c.Port = 4840
	}
	if c.TCPCount == 0 {
		c.TCPCount = DefaultTCPCount
	}
}

type CreateReq struct {
	Name             string                 `json:"name"`
	CollectorCycleMs uint64                 `json:"collectorCycleMs"`
	Variables        []*collection.Variable `json:"variables"`
	ProtocolConfig   ProtocolConfig         `json:"protocolConfig"`
}

type UpdateReq struct {
	Name             string         `json:"name"`
	CollectorCycleMs uint64         `json:"collectorCycleMs"`
	ProtocolConfig   ProtocolConfig `json:"protocolConfig"`
}

type OpcUaCollection struct {
	collection.CollectionMeta

	Location string `json:"location"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	TCPCount int    `json:"tcpCount"`

	runtime *collectorRuntime
}

func NewCollection(req *CreateReq) (*OpcUaCollection, error) {
	if err := collection.NormalizeVariables(req.Variables); err != nil {
		return nil, err
	}
	req.ProtocolConfig.Normalize()
	return &OpcUaCollection{
		CollectionMeta: collection.CollectionMeta{
			Meta:             collection.Meta{Name: req.Name},
			DriverType:       DriverType,
			DriverModel:      DriverModel,
			CollectStatus:    collection.CollectStatusStopped,
			CollectorCycleMs: req.CollectorCycleMs,
			Variables:        req.Variables,
		},
		Location: req.ProtocolConfig.Location,
		Port:     req.ProtocolConfig.Port,
		Username: req.ProtocolConfig.Username,
		Password: req.ProtocolConfig.Password,
		TCPCount: req.ProtocolConfig.TCPCount,
	}, nil
}

func (c *OpcUaCollection) ApplyUpdate(req *UpdateReq) {
	req.ProtocolConfig.Normalize()
	password := req.ProtocolConfig.Password
	if password == "" || password == "***" {
		password = c.Password
	}
	c.Name = req.Name
	c.CollectorCycleMs = req.CollectorCycleMs
	c.Location = req.ProtocolConfig.Location
	c.Port = req.ProtocolConfig.Port
	c.Username = req.ProtocolConfig.Username
	c.Password = password
	c.TCPCount = req.ProtocolConfig.TCPCount
}

type protocolConfig struct {
	Location string `json:"location"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	TCPCount int    `json:"tcpCount"`
}

func (c *OpcUaCollection) ConfigJSON() ([]byte, error) {
	tcpCount := c.TCPCount
	if tcpCount == 0 {
		tcpCount = DefaultTCPCount
	}
	return json.Marshal(protocolConfig{
		Location: c.Location,
		Port:     c.Port,
		Username: c.Username,
		Password: c.Password,
		TCPCount: tcpCount,
	})
}

func (c *OpcUaCollection) ApplyConfig(raw []byte) error {
	var config protocolConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return err
	}
	c.Location = config.Location
	c.Port = config.Port
	c.Username = config.Username
	c.Password = config.Password
	c.TCPCount = config.TCPCount
	if c.Port == 0 {
		c.Port = 4840
	}
	if c.TCPCount == 0 {
		c.TCPCount = DefaultTCPCount
	}
	return nil
}

// RedactConfig masks the password in API responses.
func (c *CreateHandler) RedactConfig(raw json.RawMessage) json.RawMessage {
	var config map[string]any
	if err := json.Unmarshal(raw, &config); err != nil {
		return raw
	}
	if password, ok := config["password"].(string); ok && password != "" {
		config["password"] = "***"
	}
	data, err := json.Marshal(config)
	if err != nil {
		return raw
	}
	return data
}

type readBatch struct {
	request   *ua.ReadRequest
	variables []*collection.Variable
}

type collectorRuntime struct {
	mu      sync.Mutex
	batches []*readBatch
	calcs   []*collection.Calculation
	pool    *clientPool

	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
}

type CollectorFactory struct{}

func (f *CollectorFactory) DriverType() string  { return DriverType }
func (f *CollectorFactory) DriverModel() string { return DriverModel }

func (f *CollectorFactory) NewCollector(meta *collection.CollectionMeta, config json.RawMessage) (collection.Collector, error) {
	item := &OpcUaCollection{}
	item.SetCollectionMeta(meta)
	if err := item.ApplyConfig(config); err != nil {
		return nil, err
	}
	return item, nil
}

func (c *OpcUaCollection) Connect(ctx context.Context) error {
	if c.runtime != nil {
		_ = c.Stop(ctx)
	}
	if c.TCPCount == 0 {
		c.TCPCount = DefaultTCPCount
	}
	if c.TCPCount < 1 || c.TCPCount > MaxTCPCount {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: tcpCount must be between 1 and %d", collection.ErrInvalidConfig, MaxTCPCount)
	}
	variables := c.GetCollectionMeta().GetVariables()
	batches, err := buildBatches(variables)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
	}
	calculations, err := collection.CompileCalculations(variables)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusError)
		return fmt.Errorf("%w: %v", collection.ErrInvalidConfig, err)
	}
	runtime := &collectorRuntime{batches: batches, calcs: calculations}
	c.runtime = runtime
	if len(batches) == 0 {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	pool, err := newClientPool(c.endpoint(), c.Username, c.Password, c.TCPCount)
	if err != nil {
		c.SetCollectStatus(collection.CollectStatusUnconnected)
		return err
	}
	if err := pool.connect(ctx); err != nil {
		pool.close(ctx)
		c.SetCollectStatus(collection.CollectStatusUnconnected)
		return err
	}
	runtime.pool = pool
	c.SetCollectStatus(collection.CollectStatusStopped)
	return nil
}

func (c *OpcUaCollection) Start(ctx context.Context) error {
	runtime := c.runtime
	if runtime == nil {
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	runtime.mu.Lock()
	if runtime.started {
		runtime.mu.Unlock()
		return nil
	}
	if len(runtime.batches) == 0 {
		runtime.mu.Unlock()
		c.SetCollectStatus(collection.CollectStatusEmptyVariable)
		return nil
	}
	if runtime.pool == nil {
		runtime.mu.Unlock()
		return fmt.Errorf("opcUa runtime is not connected")
	}
	runCtx, cancel := context.WithCancel(context.Background())
	runtime.cancel = cancel
	runtime.started = true
	runtime.wg.Add(1)
	runtime.mu.Unlock()
	go c.loop(runCtx)
	return nil
}

func (c *OpcUaCollection) Stop(ctx context.Context) error {
	runtime := c.runtime
	if runtime == nil {
		c.SetCollectStatus(collection.CollectStatusStopped)
		return nil
	}
	runtime.mu.Lock()
	cancel := runtime.cancel
	runtime.cancel = nil
	runtime.started = false
	pool := runtime.pool
	runtime.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	runtime.wg.Wait()
	if pool != nil {
		pool.close(ctx)
	}
	runtime.mu.Lock()
	if runtime.pool == pool {
		runtime.pool = nil
	}
	runtime.mu.Unlock()
	c.SetCollectStatus(collection.CollectStatusStopped)
	return nil
}

func (c *OpcUaCollection) loop(ctx context.Context) {
	runtime := c.runtime
	if runtime == nil {
		return
	}
	defer runtime.wg.Done()
	cycle := time.Duration(c.CollectorCycleMs) * time.Millisecond
	if cycle <= 0 {
		cycle = time.Second
	}
	for {
		started := time.Now()
		if !c.poll(ctx) {
			return
		}
		delay := cycle - time.Since(started)
		if delay < 0 {
			delay = 0
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

type batchResult struct {
	values []decodedValue
	err    error
}

type decodedValue struct {
	variable *collection.Variable
	value    any
}

func (c *OpcUaCollection) poll(ctx context.Context) bool {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return false
	}
	results := make(chan batchResult, len(runtime.batches))
	var wait sync.WaitGroup
	for _, batch := range runtime.batches {
		wait.Add(1)
		go func(item *readBatch) {
			defer wait.Done()
			values, err := c.readBatch(ctx, item)
			results <- batchResult{values: values, err: err}
		}(batch)
	}
	wait.Wait()
	close(results)

	failed := false
	for result := range results {
		if result.err != nil {
			failed = true
			continue
		}
		for _, item := range result.values {
			if item.variable != nil {
				c.GetCollectionMeta().SetVariableValue(item.variable.Name, item.value)
			}
		}
	}
	if ctx.Err() != nil {
		return false
	}
	if failed {
		c.SetCollectStatus(collection.CollectStatusCollectingError)
		return true
	}
	c.SetCollectStatus(collection.CollectStatusCollecting)
	if err := collection.Calculate(c.GetCollectionMeta(), runtime.calcs); err != nil {
		c.SetCollectStatus(collection.CollectStatusCollectingError)
	}
	return true
}

func (c *OpcUaCollection) readBatch(ctx context.Context, batch *readBatch) ([]decodedValue, error) {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return nil, fmt.Errorf("opcUa runtime is not connected")
	}
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		client, err := runtime.pool.acquire(ctx)
		if err != nil {
			return nil, err
		}
		response, err := client.Read(ctx, batch.request)
		runtime.pool.release(client, err != nil)
		if err != nil {
			lastErr = err
			continue
		}
		if response == nil || len(response.Results) != len(batch.variables) {
			lastErr = fmt.Errorf("opcUa response length mismatch")
			continue
		}
		values := make([]decodedValue, 0, len(batch.variables))
		for i, result := range response.Results {
			if result == nil || result.Status != ua.StatusOK || result.Value == nil {
				lastErr = fmt.Errorf("opcUa node %s read failed", batch.variables[i].Address)
				continue
			}
			values = append(values, decodedValue{
				variable: batch.variables[i],
				value:    collection.ApplyTransform(result.Value.Value(), batch.variables[i]),
			})
		}
		if lastErr != nil {
			return nil, lastErr
		}
		return values, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("opcUa batch failed")
	}
	return nil, lastErr
}

func buildBatches(variables []*collection.Variable) ([]*readBatch, error) {
	physical := make([]*collection.Variable, 0, len(variables))
	for _, variable := range variables {
		if variable == nil || variable.IsCalculated || variable.AccessMode == collection.AccessModeWrite {
			continue
		}
		physical = append(physical, variable)
	}
	if len(physical) == 0 {
		return nil, nil
	}
	batches := make([]*readBatch, 0)
	for start := 0; start < len(physical); start += 1000 {
		end := start + 1000
		if end > len(physical) {
			end = len(physical)
		}
		nodes := make([]*ua.ReadValueID, 0, end-start)
		for _, variable := range physical[start:end] {
			nodeID, err := parseNodeID(variable.Address)
			if err != nil {
				return nil, fmt.Errorf("variable %s: %w", variable.Name, err)
			}
			nodes = append(nodes, &ua.ReadValueID{NodeID: nodeID})
		}
		batches = append(batches, &readBatch{
			request: &ua.ReadRequest{
				MaxAge:             2000,
				TimestampsToReturn: ua.TimestampsToReturnBoth,
				NodesToRead:        nodes,
			},
			variables: physical[start:end],
		})
	}
	return batches, nil
}

func parseNodeID(address string) (*ua.NodeID, error) {
	value := strings.TrimSpace(address)
	if value == "" {
		return nil, fmt.Errorf("address is empty")
	}
	return ua.ParseNodeID(value)
}

func (c *OpcUaCollection) endpoint() string {
	location := strings.TrimRight(strings.TrimSpace(c.Location), "/")
	if c.Port <= 0 {
		return location
	}
	if strings.HasSuffix(location, fmt.Sprintf(":%d", c.Port)) {
		return location
	}
	return fmt.Sprintf("%s:%d", location, c.Port)
}

var _ collection.Collector = (*OpcUaCollection)(nil)
var _ collection.CollectorFactory = (*CollectorFactory)(nil)
var _ collection.ConfigRedactor = (*CreateHandler)(nil)
