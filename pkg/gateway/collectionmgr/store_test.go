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
	"harns-platform/pkg/gateway/collection/opcua"
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
	service, err := NewService(store, &modbustcp.CreateHandler{}, &opcua.CreateHandler{})
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
		"protocolConfig":{"location":"127.0.0.1:502","memoryLayout":"ABCD","positionAddress":0},
		"variables":[{
			"name":"temperature",
			"type":"telemetry",
			"dataType":"float32",
			"address":"1!40001",
			"accessMode":"r",
			"offset":0,
			"rate":0,
			"precision":2
		}]
	}`

	result, err := service.Create(context.Background(), []byte(body))
	require.NoError(t, err)
	created, ok := result.(*v1.CollectionDetailVo)
	require.True(t, ok)
	require.NotEmpty(t, created.ID)
	assert.Equal(t, "modbus-tcp-1", created.Name)
	assert.Equal(t, modbustcp.DriverType, created.DriverType)
	assert.Equal(t, modbustcp.DriverModel, created.DriverModel)
	assert.Equal(t, uint64(1000), created.CollectorCycleMs)
	assert.Contains(t, string(created.ProtocolConfig), `"location":"127.0.0.1:502"`)

	record, err := service.Get(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, "modbus-tcp-1", record.Name)
	assert.Equal(t, "127.0.0.1:502", record.ProtocolConfigValue().Location)
	assert.Len(t, record.Variables, 1)
	assert.Equal(t, "temperature", record.Variables[0].Name)

	page, err := service.List(context.Background(), v1.CollectionQuery{PageQuery: apis.PageQuery{Page: 1, PageSize: 10}})
	require.NoError(t, err)
	require.Len(t, page.List, 1)
	assert.Equal(t, int64(1), page.List[0].VariableCount)
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
		"collectorCycleMs":1000,"protocolConfig":{"location":"127.0.0.1:502","memoryLayout":"ABCD"},
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
		"collectorCycleMs":1000,"protocolConfig":{"location":"127.0.0.1:70000","memoryLayout":"ABCD"},"variables":[]
	}`
	_, err := service.Create(ctx, []byte(body))
	require.Error(t, err)
	var fieldErr *validation.ValidationError
	require.True(t, errors.As(err, &fieldErr))
	assert.Equal(t, "protocolConfig.location", fieldErr.Issues[0].Field)

	body = `{
		"driverType":"modbus","driverModel":"modbusTcp","name":"bad-schema",
		"collectorCycleMs":1000,"protocolConfig":{"location":"127.0.0.1:502","memoryLayout":"ZZZZ"},"variables":[]
	}`
	_, err = service.Create(ctx, []byte(body))
	require.Error(t, err)
	require.True(t, errors.As(err, &fieldErr))
	assert.Equal(t, "protocolConfig.memoryLayout", fieldErr.Issues[0].Field)

	body = `{
		"driverType":"modbus","driverModel":"modbusTcp","name":"bad-schema",
		"collectorCycleMs":1000,"protocolConfig":{"location":"127.0.0.1:502","backupLocation":"bad","memoryLayout":"ABCD"},"variables":[]
	}`
	_, err = service.Create(ctx, []byte(body))
	require.Error(t, err)
	require.True(t, errors.As(err, &fieldErr))
	assert.Equal(t, "protocolConfig.backupLocation", fieldErr.Issues[0].Field)

	body = `{
		"driverType":"modbus","driverModel":"modbusTcp","name":"bad-schema",
		"collectorCycleMs":1000,"protocolConfig":{"location":"127.0.0.1:502","memoryLayout":"ABCD","tcpCount":4},"variables":[]
	}`
	_, err = service.Create(ctx, []byte(body))
	require.Error(t, err)
	require.True(t, errors.As(err, &fieldErr))
	assert.Equal(t, "protocolConfig.tcpCount", fieldErr.Issues[0].Field)

	body = `{
		"driverType":"modbus","driverModel":"modbusTcp","name":"bad-schema",
		"collectorCycleMs":1000,"protocolConfig":{"location":"127.0.0.1:502","memoryLayout":"ABCD"},
		"variables":[{"name":"temperature","dataType":"float32","address":"40001","accessMode":"r"}]
	}`
	_, err = service.Create(ctx, []byte(body))
	require.Error(t, err)
	require.True(t, errors.As(err, &fieldErr))
	assert.Equal(t, "variables[0].address", fieldErr.Issues[0].Field)
}

func TestCreateCollectionPersistsBits(t *testing.T) {
	service, _ := newTestService(t)
	body := `{
		"driverType":"modbus","driverModel":"modbusTcp","name":"bits-test",
		"collectorCycleMs":1000,"protocolConfig":{"location":"127.0.0.1:502","memoryLayout":"ABCD"},
		"variables":[{"name":"flag","dataType":"bool","address":"1!40001","bits":3,"accessMode":"r"}]
	}`
	result, err := service.Create(context.Background(), []byte(body))
	require.NoError(t, err)
	created := result.(*v1.CollectionDetailVo)
	record, err := service.Get(context.Background(), created.ID)
	require.NoError(t, err)
	require.Len(t, record.Variables, 1)
	assert.Equal(t, uint8(3), record.Variables[0].BitsValue())
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
		"collectorCycleMs":1000,"protocolConfig":{"location":"127.0.0.1:502","memoryLayout":"ABCD"},
		"variables":[{"name":"temperature","dataType":"float32","address":"1!40001","accessMode":"r"}]
	}`
	result, err := service.Create(context.Background(), []byte(body))
	require.NoError(t, err)
	created := result.(*v1.CollectionDetailVo)
	current, ok := runtime.Get(created.ID)
	require.True(t, ok)
	assert.Equal(t, collection.CollectStatusUnconnected, current.Collector.GetCollectStatus())

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

func TestOpcUaPasswordIsRedactedFromDetail(t *testing.T) {
	service, _ := newTestService(t)
	body := `{
		"driverType":"opcUa","driverModel":"opcUa","name":"ua-test","collectorCycleMs":1000,
		"protocolConfig":{"location":"opc.tcp://127.0.0.1","port":4840,"username":"u","password":"secret","tcpCount":1},
		"variables":[{"name":"temperature","dataType":"number","address":"ns=2;s=Tag","accessMode":"r"}]
	}`
	result, err := service.Create(context.Background(), []byte(body))
	require.NoError(t, err)
	created := result.(*v1.CollectionDetailVo)
	assert.Contains(t, string(created.ProtocolConfig), `"password":"***"`)
	assert.NotContains(t, string(created.ProtocolConfig), "secret")

	record, err := service.Get(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Contains(t, string(record.ProtocolConfig), `"password":"secret"`)
}

func TestCollectionEnableLifecycle(t *testing.T) {
	service, _ := newTestService(t)
	body := `{
		"driverType":"modbus","driverModel":"modbusTcp","name":"enable-test",
		"collectorCycleMs":1000,"protocolConfig":{"location":"127.0.0.1:502","memoryLayout":"ABCD"},
		"variables":[{"name":"temperature","dataType":"float32","address":"1!40001","accessMode":"r"}]
	}`
	result, err := service.Create(context.Background(), []byte(body))
	require.NoError(t, err)
	created := result.(*v1.CollectionDetailVo)
	assert.True(t, created.Enable)

	enabled, err := service.SetEnable(context.Background(), created.ID, false)
	require.NoError(t, err)
	assert.False(t, enabled.Enable)

	_, err = service.Start(context.Background(), created.ID)
	require.Error(t, err)
	var validationErr *validation.ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "enable", validationErr.Issues[0].Field)

	enabled, err = service.SetEnable(context.Background(), created.ID, true)
	require.NoError(t, err)
	assert.True(t, enabled.Enable)
}

func TestStoreMigratesVariablePresenceColumns(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`PRAGMA foreign_keys = OFF`)
	require.NoError(t, err)

	_, err = db.Exec(`CREATE TABLE collection_variables (
		collection_id TEXT NOT NULL,
		name TEXT NOT NULL,
		variable_type TEXT NOT NULL,
		data_type TEXT NOT NULL,
		address TEXT NOT NULL DEFAULT '',
		bits INTEGER NOT NULL DEFAULT 0,
		access_mode TEXT NOT NULL,
		is_calculated INTEGER NOT NULL DEFAULT 0,
		expression TEXT NOT NULL DEFAULT '',
		value_offset REAL NOT NULL DEFAULT 0,
		rate REAL NOT NULL DEFAULT 1,
		precision INTEGER NOT NULL DEFAULT 0,
		bits_set INTEGER NOT NULL DEFAULT 0,
		rate_set INTEGER NOT NULL DEFAULT 0,
		offset_set INTEGER NOT NULL DEFAULT 0,
		precision_set INTEGER NOT NULL DEFAULT 0,
		sort INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (collection_id, name)
	)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO collection_variables (
		collection_id, name, variable_type, data_type, address, bits, access_mode,
		is_calculated, expression, value_offset, rate, precision, sort,
		bits_set, rate_set, offset_set, precision_set
	) VALUES ('c1', 'flag', 'telemetry', 'BOOL', '1!40001', 3, 'r', 0, '', 0, 1, 2, 0, 1, 0, 0, 1)`)
	require.NoError(t, err)

	store := NewStore(db)
	require.NoError(t, store.Initialize(context.Background()))
	variables, total, err := store.ListVariables(context.Background(), "c1", apis.PageQuery{Page: 1, PageSize: 10}, "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, variables, 1)
	assert.Equal(t, uint8(3), variables[0].BitsValue())
	assert.Nil(t, variables[0].Rate)
	assert.Nil(t, variables[0].Offset)
	assert.Equal(t, 2, variables[0].PrecisionValue())

	columns, err := store.variableColumnInfo(context.Background())
	require.NoError(t, err)
	_, hasBitsSet := columns["bits_set"]
	assert.False(t, hasBitsSet)
}

type modbusRuntimeFactory struct {
	collectors []*fakeCollector
}

func (f *modbusRuntimeFactory) DriverType() string  { return "modbus" }
func (f *modbusRuntimeFactory) DriverModel() string { return "modbusTcp" }
func (f *modbusRuntimeFactory) NewCollector(meta *collection.CollectionMeta, _ json.RawMessage) (collection.Collector, error) {
	meta.IndexCollection()
	collector := &fakeCollector{meta: meta}
	f.collectors = append(f.collectors, collector)
	return collector, nil
}

func TestAddVariablesRefreshesRunningRuntime(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	driverStore := drivermgr.NewStore(db)
	require.NoError(t, driverStore.Initialize(context.Background()))
	store := NewStore(db)
	require.NoError(t, store.Initialize(context.Background()))
	service, err := NewService(store, &modbustcp.CreateHandler{})
	require.NoError(t, err)
	factory := &modbusRuntimeFactory{}
	runtime, err := NewRuntimeManager(factory)
	require.NoError(t, err)
	service.SetRuntimeManager(runtime)
	t.Cleanup(func() { _ = runtime.Close(context.Background()) })

	body := `{
		"driverType":"modbus","driverModel":"modbusTcp","name":"add-vars",
		"collectorCycleMs":1000,"protocolConfig":{"location":"127.0.0.1:502","memoryLayout":"ABCD"},
		"variables":[{"name":"first","dataType":"int16","address":"1!40001","accessMode":"r"}]
	}`
	result, err := service.Create(context.Background(), []byte(body))
	require.NoError(t, err)
	created, ok := result.(*v1.CollectionDetailVo)
	require.True(t, ok)
	require.Len(t, factory.collectors, 1)
	assert.Len(t, factory.collectors[0].meta.GetVariables(), 1)

	_, err = service.AddVariables(context.Background(), created.ID, []*collection.Variable{{
		Name: "second", DataType: "int16", Address: "1!40001", AccessMode: collection.AccessModeRead,
	}})
	require.NoError(t, err)
	require.Len(t, factory.collectors, 2)
	assert.Len(t, factory.collectors[1].meta.GetVariables(), 2)
	assert.Equal(t, 1, factory.collectors[0].stops)
}

func TestCreateCollectionRejectsMissingCalculatedReference(t *testing.T) {
	service, _ := newTestService(t)
	body := `{
		"driverType":"modbus","driverModel":"modbusTcp","name":"bad-calc",
		"collectorCycleMs":1000,"protocolConfig":{"location":"127.0.0.1:502","memoryLayout":"ABCD"},
		"variables":[
			{"name":"one","dataType":"float64","address":"1!40001","accessMode":"r"},
			{"name":"result","dataType":"float64","address":"","accessMode":"r","isCalculated":true,"expression":"one + three"}
		]
	}`
	_, err := service.Create(context.Background(), []byte(body))
	require.Error(t, err)
	var verr *validation.ValidationError
	require.ErrorAs(t, err, &verr)
	require.Len(t, verr.Issues, 1)
	assert.Equal(t, "variables[1].expression", verr.Issues[0].Field)
	assert.Equal(t, "validate.calculated_variable_reference_missing", verr.Issues[0].MsgID)
}

func TestDeleteReferencedCalculatedVariableRejected(t *testing.T) {
	service, store := newTestService(t)
	body := `{
		"driverType":"modbus","driverModel":"modbusTcp","name":"calc-delete",
		"collectorCycleMs":1000,"protocolConfig":{"location":"127.0.0.1:502","memoryLayout":"ABCD"},
		"variables":[
			{"name":"temperature","dataType":"float64","address":"1!40001","accessMode":"r"},
			{"name":"energy","dataType":"float64","address":"","accessMode":"r","isCalculated":true,"expression":"temperature * 2"}
		]
	}`
	result, err := service.Create(context.Background(), []byte(body))
	require.NoError(t, err)
	created, ok := result.(*v1.CollectionDetailVo)
	require.True(t, ok)

	_, err = service.DeleteVariables(context.Background(), created.ID, []string{"temperature"})
	require.Error(t, err)
	var verr *validation.ValidationError
	require.ErrorAs(t, err, &verr)
	assert.Equal(t, "validate.calculated_variable_reference_missing", verr.Issues[0].MsgID)

	record, err := store.Get(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Len(t, record.Variables, 2)
}
