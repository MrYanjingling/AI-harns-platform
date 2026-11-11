package s7

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/utils/binutil"
)

func TestParseAddress(t *testing.T) {
	cases := map[string]parsedAddress{
		"DB1.DBW0":   {area: areaDB, blockSize: 1, address: 0},
		"DB1.DBX0.5": {area: areaDB, blockSize: 1, address: 0, bit: 5},
		"I0.0":       {area: areaI, address: 0},
		"Q10.3":      {area: areaQ, address: 10, bit: 3},
		"M100":       {area: areaM, address: 100},
	}
	for value, expected := range cases {
		parsed, err := parseAddress(value)
		require.NoError(t, err, value)
		assert.Equal(t, expected, parsed, value)
	}
}

func TestBuildAndDecodeFrame(t *testing.T) {
	variables := []*collection.Variable{{
		Name: "speed", DataType: "int16", Address: "DB1.DBW0", AccessMode: collection.AccessModeRead,
	}}
	frames, err := buildFrames(variables, 240)
	require.NoError(t, err)
	require.Len(t, frames, 1)
	assert.Equal(t, 6, frames[0].DataLength)

	values, err := decodeFrame(frames[0], []byte{0, 0, 0, 0, 0x00, 0x2A})
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.Equal(t, int16(42), values[0].value)
}

func TestBuildFramesRejectsString(t *testing.T) {
	_, err := buildFrames([]*collection.Variable{{
		Name: "text", DataType: "string", Address: "DB1.DBB0", AccessMode: collection.AccessModeRead,
	}}, 240)
	require.Error(t, err)
}

func TestSessionConfiguration(t *testing.T) {
	cotp := newCOTPConnectMessage(0, 1, 2)
	require.Len(t, cotp, 22)
	assert.Equal(t, uint8(2), cotp[20])
	assert.Equal(t, uint8(1), cotp[21])

	setup := newS7COMMSetupMessage(960)
	require.Len(t, setup, 25)
	assert.Equal(t, uint16(960), binutil.ParseUint16(setup[23:]))
}

func TestProtocolConfigDefaultsAndStringValues(t *testing.T) {
	created, err := NewCollection(&CreateReq{
		Name:             "s7-default",
		CollectorCycleMs: 1000,
		ProtocolConfig:   ProtocolConfig{Location: "127.0.0.1"},
	})
	require.NoError(t, err)
	assert.Equal(t, DefaultPDUSize, created.PDUSize)
	assert.Equal(t, uint8(DefaultConnectionType), created.ConnectionType)

	created = &S7Collection{}
	require.NoError(t, created.ApplyConfig([]byte(`{"location":"127.0.0.1","pduSize":"960","connectionType":"3"}`)))
	assert.Equal(t, 960, created.PDUSize)
	assert.Equal(t, uint8(3), created.ConnectionType)

	config, err := created.ConfigJSON()
	require.NoError(t, err)
	assert.Contains(t, string(config), `"pduSize":960`)
	assert.Contains(t, string(config), `"connectionType":3`)
}

func TestNewS7FrameTPKTLength(t *testing.T) {
	items := make([]*frameItem, 38)
	for i := range items {
		items[i] = &frameItem{RequestData: make([]byte, 12)}
	}
	frame := newS7Frame(areaDB, nil, items, 0)
	require.Len(t, frame.Request, 475)
	assert.Equal(t, uint16(475), binutil.ParseUint16(frame.Request[2:]))
}

func TestParseAddressArrayLength(t *testing.T) {
	cases := map[string]parsedAddress{
		"DB1.DBB0.10": {area: areaDB, blockSize: 1, address: 0, length: 10},
		"DB1.DBW0.10": {area: areaDB, blockSize: 1, address: 0, length: 10},
		"DB1.DBD0.10": {area: areaDB, blockSize: 1, address: 0, length: 10},
		"DB1.DBX0.0":  {area: areaDB, blockSize: 1, address: 0},
		"I0":          {area: areaI, address: 0},
		"IW0":         {area: areaI, address: 0},
		"I0.0":        {area: areaI, address: 0},
	}
	for value, expected := range cases {
		parsed, err := parseAddress(value)
		require.NoError(t, err, value)
		assert.Equal(t, expected, parsed, value)
	}
}

func TestBuildAndDecodeStringAndByteArray(t *testing.T) {
	variables := []*collection.Variable{
		{Name: "text", DataType: "STRING", Address: "DB1.DBW0.10", AccessMode: collection.AccessModeRead},
		{Name: "bytes", DataType: "BYTE_ARRAY", Address: "DB1.DBW20.4", AccessMode: collection.AccessModeRead},
	}
	frames, err := buildFrames(variables, 480)
	require.NoError(t, err)
	require.Len(t, frames, 1)
	require.Len(t, frames[0].Variables, 2)
	assert.Equal(t, uint16(10), frames[0].Variables[0].Length)
	assert.Equal(t, uint16(4), frames[0].Variables[1].Length)
	assert.Equal(t, 22, frames[0].DataLength)

	payload := make([]byte, 22)
	copy(payload[4:14], []byte("hello\x00\x00\x00\x00\x00"))
	copy(payload[18:22], []byte{1, 2, 3, 255})
	values, err := decodeFrame(frames[0], payload)
	require.NoError(t, err)
	assert.Equal(t, "hello", values[0].value)
	assert.Equal(t, []int{1, 2, 3, 255}, values[1].value)
}

func TestBuildFramesRejectsInvalidArrayLength(t *testing.T) {
	_, err := buildFrames([]*collection.Variable{{
		Name: "text", DataType: "STRING", Address: "DB1.DBW0", AccessMode: collection.AccessModeRead,
	}}, 480)
	require.Error(t, err)

	_, err = buildFrames([]*collection.Variable{{
		Name: "value", DataType: "INT16", Address: "DB1.DBW0.10", AccessMode: collection.AccessModeRead,
	}}, 480)
	require.Error(t, err)
}
