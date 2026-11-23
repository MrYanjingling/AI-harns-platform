package drivermgr

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/drivermgr/v1"
	"harns-platform/pkg/gateway/sqlite"
)

func newTestService(t *testing.T) (*Service, *Store) {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)
	require.NoError(t, store.Initialize(context.Background()))
	return NewService(store), store
}

func TestInitializeIsIdempotentAndSeedsBuiltins(t *testing.T) {
	service, store := newTestService(t)
	require.NoError(t, store.Initialize(context.Background()))

	items, err := service.List(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, items, len(BuiltinDrivers))
	for _, item := range items {
		assert.True(t, item.BuiltinDriver)
		assert.NotEmpty(t, item.PluginName)
		assert.NotContains(t, item.PluginName, "_")
		schema, err := ParseSchema(item.Schema)
		require.NoError(t, err)
		assert.NotEmpty(t, schema.Fields)
		assert.NotNil(t, schema.VariableAddress)
		assert.NotEmpty(t, schema.VariableDataTypes)
	}
}

func TestCreateUpdateDeleteCustomDriver(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()

	created, err := service.Create(ctx, &v1.CreateDriverReq{
		DriverType: "modbus",
		Model:      "customTcp",
		Schema:     json.RawMessage(`{"type":"object","properties":{"host":{"type":"string"}}}`),
		PluginName: "custom-tcp",
	})
	require.NoError(t, err)
	assert.False(t, created.BuiltinDriver)
	assert.NotEmpty(t, created.ID)

	updated, err := service.Update(ctx, created.ID, &v1.UpdateDriverReq{
		DriverType: "modbus",
		Model:      "customTcpV2",
		Schema:     json.RawMessage(`{"type":"object"}`),
		PluginName: "custom-tcp-v2",
	})
	require.NoError(t, err)
	assert.Equal(t, "customTcpV2", updated.Model)
	assert.Equal(t, "custom-tcp-v2", updated.PluginName)

	require.NoError(t, service.Delete(ctx, created.ID))
	_, err = service.Get(ctx, created.ID)
	require.Error(t, err)
	assert.Equal(t, int(response.ErrCodeDriverNotFound), response.ErrorCode(err))
}

func TestCreateRejectsDuplicateAndInvalidValues(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()

	_, err := service.Create(ctx, &v1.CreateDriverReq{
		DriverType: "modbus",
		Model:      "modbusTcp",
		PluginName: "another-modbus-tcp",
	})
	require.Error(t, err)
	var validation *ValidationError
	require.True(t, errors.As(err, &validation))
	assert.Equal(t, "model", validation.Issues[0].Field)

	_, err = service.Create(ctx, &v1.CreateDriverReq{
		DriverType: "modbus",
		Model:      "custom",
		PluginName: "bad/name",
	})
	require.Error(t, err)
	require.True(t, errors.As(err, &validation))
	assert.Equal(t, "pluginName", validation.Issues[0].Field)

	_, err = service.Create(ctx, &v1.CreateDriverReq{
		DriverType: "modbus",
		Model:      "custom",
		Schema:     json.RawMessage(`[]`),
		PluginName: "custom",
	})
	require.Error(t, err)
	require.True(t, errors.As(err, &validation))
	assert.Equal(t, "schema", validation.Issues[0].Field)
}

func TestCreateRejectsDuplicateModelAndPluginName(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()

	created, err := service.Create(ctx, &v1.CreateDriverReq{
		DriverType: "custom-a",
		Model:      "shared-model",
		PluginName: "shared-model-plugin",
	})
	require.NoError(t, err)

	_, err = service.Create(ctx, &v1.CreateDriverReq{
		DriverType: "custom-b",
		Model:      "shared-model",
		PluginName: "another-plugin",
	})
	require.Error(t, err)
	var validation *ValidationError
	require.True(t, errors.As(err, &validation))
	assert.Equal(t, "model", validation.Issues[0].Field)

	_, err = service.Create(ctx, &v1.CreateDriverReq{
		DriverType: "custom-b",
		Model:      "another-model",
		PluginName: "shared-model-plugin",
	})
	require.Error(t, err)
	require.True(t, errors.As(err, &validation))
	assert.Equal(t, "pluginName", validation.Issues[0].Field)

	_, err = service.Update(ctx, created.ID, &v1.UpdateDriverReq{
		DriverType: "custom-a",
		Model:      "modbusTcp",
		PluginName: "shared-model-plugin",
	})
	require.Error(t, err)
	require.True(t, errors.As(err, &validation))
	assert.Equal(t, "model", validation.Issues[0].Field)
}

func TestBuiltinDriverIsReadOnly(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()

	_, err := service.Update(ctx, BuiltinModbusTCPID, &v1.UpdateDriverReq{
		DriverType: "modbus",
		Model:      "changed",
		PluginName: "changed",
	})
	require.Error(t, err)
	assert.Equal(t, int(response.ErrCodeDriverBuiltinReadOnly), response.ErrorCode(err))

	err = service.Delete(ctx, BuiltinModbusTCPID)
	require.Error(t, err)
	assert.Equal(t, int(response.ErrCodeDriverBuiltinReadOnly), response.ErrorCode(err))
}

func TestListFiltersByDriverType(t *testing.T) {
	service, _ := newTestService(t)
	items, err := service.List(context.Background(), DriverTypeS7)
	require.NoError(t, err)
	require.Len(t, items, 4)
	models := make([]string, 0, len(items))
	for _, item := range items {
		models = append(models, item.Model)
	}
	assert.ElementsMatch(t, []string{ModelS71500, ModelS7200, ModelMPI, ModelPPI}, models)
}

func TestListGroupedAndFuzzyDriverType(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()

	groups, err := service.ListGrouped(ctx, "mod")
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, DriverTypeModbus, groups[0].DriverType)
	assert.Len(t, groups[0].Models, 4)

	items, err := service.List(ctx, "MOD")
	require.NoError(t, err)
	require.Len(t, items, 4)

	items, err = service.List(ctx, "%")
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestUploadPlugin(t *testing.T) {
	service, store := newTestService(t)
	service.pluginDir = t.TempDir()
	created, err := service.Create(context.Background(), &v1.CreateDriverReq{
		DriverType: "modbus",
		Model:      "plugin-target",
		PluginName: "plugin-target",
		Schema:     json.RawMessage(`{"fields":[]}`),
	})
	require.NoError(t, err)

	updated, err := service.UploadPlugin(context.Background(), created.ID, "libdriver.so", strings.NewReader("binary"))
	require.NoError(t, err)
	assert.Equal(t, "plugin-target", updated.PluginName)
	data, err := os.ReadFile(filepath.Join(service.pluginDir, "plugin-target", "linux-"+runtime.GOARCH, "plugin-target.so"))
	require.NoError(t, err)
	assert.Equal(t, "binary", string(data))

	_, err = service.UploadPlugin(context.Background(), created.ID, "libdriver.txt", strings.NewReader("bad"))
	require.Error(t, err)

	builtin, err := store.GetByModel(context.Background(), "modbusTcp")
	require.NoError(t, err)
	_, err = service.UploadPlugin(context.Background(), builtin.ID, "libdriver.so", strings.NewReader("bad"))
	require.Error(t, err)
}
