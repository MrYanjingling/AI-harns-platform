package custom

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/pluginmgr"
)

type fakeDriver struct {
	frame pluginmgr.ReadFrame
}

func (d *fakeDriver) Descriptor() pluginmgr.Descriptor {
	return pluginmgr.Descriptor{ABIVersion: pluginmgr.ABIVersion}
}
func (d *fakeDriver) ValidateConfig(json.RawMessage) error { return nil }
func (d *fakeDriver) TransportKind(json.RawMessage) (pluginmgr.TransportKind, error) {
	return pluginmgr.TransportTCP, nil
}
func (d *fakeDriver) NewTransport(context.Context, json.RawMessage) (pluginmgr.Transport, error) {
	return &fakeTransport{}, nil
}
func (d *fakeDriver) BuildReadFrames(variables []*collection.Variable, _ json.RawMessage) ([]pluginmgr.ReadFrame, error) {
	d.frame = pluginmgr.ReadFrame{Request: []byte{1}, Variables: variables}
	return []pluginmgr.ReadFrame{d.frame}, nil
}
func (d *fakeDriver) DecodeRead(frame pluginmgr.ReadFrame, response []byte) ([]pluginmgr.DecodedValue, error) {
	return []pluginmgr.DecodedValue{{Variable: frame.Variables[0], Value: int16(response[0])}}, nil
}
func (d *fakeDriver) BuildWrite(collection.ActionValue, json.RawMessage) ([]byte, error) {
	return []byte{2}, nil
}
func (d *fakeDriver) DecodeWrite([]byte) error { return nil }

type fakeTransport struct{ closed bool }

func (t *fakeTransport) Exchange(context.Context, []byte) ([]byte, error) { return []byte{42}, nil }
func (t *fakeTransport) Available() bool                                  { return !t.closed }
func (t *fakeTransport) Close() error                                     { t.closed = true; return nil }

func TestCustomCollectorPoll(t *testing.T) {
	variable := &collection.Variable{Name: "temperature", DataType: "int16", AccessMode: collection.AccessModeRead}
	meta := &collection.CollectionMeta{Variables: []*collection.Variable{variable}}
	driver := &fakeDriver{}
	item := NewCollector(meta, driver, json.RawMessage(`{"concurrency":1}`))
	t.Cleanup(func() { _ = item.Stop(context.Background()) })
	require.NoError(t, item.Connect(context.Background()))
	require.True(t, item.poll(context.Background()))
	value, ok := item.GetVariableValue("temperature")
	require.True(t, ok)
	assert.Equal(t, int16(42), value)
	assert.Equal(t, collection.CollectStatusCollecting, item.GetCollectStatus())
}
