package collectionmgr

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/apis"
	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/modbustcp"
	v1 "harns-platform/pkg/gateway/collectionmgr/v1"
	"harns-platform/pkg/gateway/drivermgr"
	"harns-platform/pkg/gateway/sqlite"
	"harns-platform/pkg/gateway/validation"
)

func newTestService(t *testing.T) (*Service, *Store) {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	driverStore := drivermgr.NewStore(db)
	require.NoError(t, driverStore.Initialize(context.Background()))

	store := NewStore(db)
	require.NoError(t, store.Initialize(context.Background()))
	service, err := NewService(store, &modbustcp.CreateHandler{})
	require.NoError(t, err)
	return service, store
}

func TestCreateModbusTcpCollection(t *testing.T) {
	service, _ := newTestService(t)
	body := `{
		"driverType":"modbus",
		"driverModel":"modbusTcp",
		"name":"modbus-tcp-1",
		"collectorCycleMs":1000,
		"location":"127.0.0.1:502",
		"memoryLayout":"ABCD",
		"positionAddress":0,
		"variables":[{
			"name":"temperature",
			"type":"telemetry",
			"dataType":"float32",
			"address":"holding:40001",
			"accessMode":"r",
			"offset":0,
			"rate":0,
			"precision":2
		}]
	}`

	result, err := service.Create(context.Background(), []byte(body))
	require.NoError(t, err)
	created, ok := result.(*modbustcp.ModbusTcpCollection)
	require.True(t, ok)
	require.NotEmpty(t, created.ID)
	assert.Equal(t, "modbus-tcp-1", created.Name)
	assert.Equal(t, modbustcp.DriverType, created.DriverType)
	assert.Equal(t, modbustcp.DriverModel, created.DriverModel)
	assert.Equal(t, uint64(1000), created.CollectorCycleMs)
	assert.Equal(t, "127.0.0.1:502", created.Location)
	require.Len(t, created.Variables, 1)
	assert.Equal(t, float64(1), created.Variables[0].Rate)

	record, err := service.Get(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, "modbus-tcp-1", record.Name)
	assert.Equal(t, "127.0.0.1:502", record.ProtocolConfigValue().Location)
	assert.Len(t, record.Variables, 1)
	assert.Equal(t, "temperature", record.Variables[0].Name)
}

func TestCreateCollectionValidation(t *testing.T) {
	service, _ := newTestService(t)

	_, err := service.Create(context.Background(), []byte(`{"driverType":"modbus","driverModel":"unknown"}`))
	require.Error(t, err)
	var unsupported *validation.ValidationError
	require.True(t, errors.As(err, &unsupported))
	assert.Equal(t, "driverType", unsupported.Issues[0].Field)

	body := `{
		"driverType":"modbus","driverModel":"modbusTcp","name":"bad",
		"collectorCycleMs":1000,"location":"127.0.0.1:502",
		"memoryLayout":"ABCD",
		"variables":[{"name":"temperature","dataType":"float32","address":"","accessMode":"r"}]
	}`
	_, err = service.Create(context.Background(), []byte(body))
	require.Error(t, err)
}

func (r *CollectionRecord) ProtocolConfigValue() modbustcpConfig {
	var value modbustcpConfig
	_ = json.Unmarshal(r.ProtocolConfig, &value)
	return value
}

type modbustcpConfig struct {
	Location string `json:"location"`
}

func TestCreateCollectionUsesDriverSchema(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()

	body := `{
		"driverType":"modbus","driverModel":"modbusTcp","name":"bad-schema",
		"collectorCycleMs":1000,"location":"127.0.0.1:70000","memoryLayout":"ABCD","variables":[]
	}`
	_, err := service.Create(ctx, []byte(body))
	require.Error(t, err)
	var fieldErr *validation.ValidationError
	require.True(t, errors.As(err, &fieldErr))
	assert.Equal(t, "location", fieldErr.Issues[0].Field)

	body = `{
		"driverType":"modbus","driverModel":"modbusTcp","name":"bad-schema",
		"collectorCycleMs":1000,"location":"127.0.0.1:502","memoryLayout":"ZZZZ","variables":[]
	}`
	_, err = service.Create(ctx, []byte(body))
	require.Error(t, err)
	require.True(t, errors.As(err, &fieldErr))
	assert.Equal(t, "memoryLayout", fieldErr.Issues[0].Field)

	body = `{
		"driverType":"modbus","driverModel":"modbusTcp","name":"bad-schema",
		"collectorCycleMs":1000,"location":"127.0.0.1:502","backupLocation":"bad",
		"memoryLayout":"ABCD","variables":[]
	}`
	_, err = service.Create(ctx, []byte(body))
	require.Error(t, err)
	require.True(t, errors.As(err, &fieldErr))
	assert.Equal(t, "backupLocation", fieldErr.Issues[0].Field)
}

func TestCollectionServiceRuntimeIntegration(t *testing.T) {
	service, _ := newTestService(t)
	runtime, err := NewRuntimeManager(&modbustcp.CollectorFactory{})
	require.NoError(t, err)
	service.SetRuntimeManager(runtime)
	require.NoError(t, service.StartRuntime(context.Background()))
	t.Cleanup(func() { _ = runtime.Close(context.Background()) })

	body := `{
		"driverType":"modbus","driverModel":"modbusTcp","name":"runtime-test",
		"collectorCycleMs":1000,"location":"127.0.0.1:502","memoryLayout":"ABCD",
		"variables":[{"name":"temperature","dataType":"float32","address":"holding:1","accessMode":"r"}]
	}`
	result, err := service.Create(context.Background(), []byte(body))
	require.NoError(t, err)
	created := result.(*modbustcp.ModbusTcpCollection)
	current, ok := runtime.Get(created.ID)
	require.True(t, ok)
	assert.Equal(t, collection.CollectStatusError, current.Collector.GetCollectStatus())

	current.Collector.GetCollectionMeta().SetVariableValue("temperature", float64(26.5))
	page, err := service.ListVariables(context.Background(), created.ID, v1.VariableQuery{PageQuery: apis.PageQuery{Page: 1, PageSize: 10}})
	require.NoError(t, err)
	require.Len(t, page.List, 1)
	assert.Equal(t, float64(26.5), page.List[0].Value)

	_, err = service.Delete(context.Background(), created.ID)
	require.NoError(t, err)
	_, ok = runtime.Get(created.ID)
	assert.False(t, ok)
}
