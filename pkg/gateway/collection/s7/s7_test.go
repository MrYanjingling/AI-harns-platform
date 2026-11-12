package s7

import (
	"encoding/json"
	"net"
	"strconv"
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
		"V0.0":       {area: areaDB, blockSize: 1, address: 0},
		"V0.5":       {area: areaDB, blockSize: 1, address: 0, bit: 5},
		"VB10":       {area: areaDB, blockSize: 1, address: 10},
		"VW20":       {area: areaDB, blockSize: 1, address: 20},
		"VD30":       {area: areaDB, blockSize: 1, address: 30},
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

	s7200 := newS7200COTPConnectMessage(0x1000, 0x1000)
	require.Len(t, s7200, 22)
	assert.Equal(t, []byte{0xc1, 0x02, 0x10, 0x00, 0xc2, 0x02, 0x10, 0x00}, s7200[14:])

	custom := newS7200COTPConnectMessage(0x1234, 0xABCD)
	assert.Equal(t, []byte{0xc1, 0x02, 0x12, 0x34, 0xc2, 0x02, 0xAB, 0xCD}, custom[14:])

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
	frame := newS7Frame(items, 0)
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

func TestBuildFramesMergesNearbyTags(t *testing.T) {
	variables := []*collection.Variable{
		{Name: "a", DataType: "int16", Address: "DB1.DBW0", AccessMode: collection.AccessModeRead},
		{Name: "b", DataType: "int16", Address: "DB1.DBW2", AccessMode: collection.AccessModeRead},
	}
	frames, err := buildFrames(variables, 480)
	require.NoError(t, err)
	require.Len(t, frames, 1)
	assert.Equal(t, 8, frames[0].DataLength)
	require.Len(t, frames[0].Variables, 2)
	assert.Equal(t, uint(4), frames[0].Variables[0].Start)
	assert.Equal(t, uint(6), frames[0].Variables[1].Start)
}

func TestBuildFramesDoesNotMergeStringWithNumeric(t *testing.T) {
	variables := []*collection.Variable{
		{Name: "text", DataType: "STRING", Address: "DB1.DBW0.10", AccessMode: collection.AccessModeRead},
		{Name: "value", DataType: "int16", Address: "DB1.DBW10", AccessMode: collection.AccessModeRead},
	}
	frames, err := buildFrames(variables, 480)
	require.NoError(t, err)
	require.Len(t, frames, 1)
	assert.Equal(t, 20, frames[0].DataLength)
	require.Len(t, frames[0].Variables, 2)
	assert.Equal(t, uint(4), frames[0].Variables[0].Start)
	assert.Equal(t, uint(18), frames[0].Variables[1].Start)
}

func TestBuildFramesSplitsByPDUBudget(t *testing.T) {
	variables := []*collection.Variable{
		{Name: "a", DataType: "STRING", Address: "DB1.DBW0.10", AccessMode: collection.AccessModeRead},
		{Name: "b", DataType: "STRING", Address: "DB1.DBW20.10", AccessMode: collection.AccessModeRead},
	}
	frames, err := buildFrames(variables, 47)
	require.NoError(t, err)
	require.Len(t, frames, 2)
}

func TestPackItemsUsesPDUBudgetInsteadOfFixed27(t *testing.T) {
	variables := make([]*collection.Variable, 0, 30)
	for i := 0; i < 30; i++ {
		variables = append(variables, &collection.Variable{
			Name:       "v" + string(rune('a'+i)),
			DataType:   "STRING",
			Address:    "DB1.DBW" + strconv.Itoa(i*40) + ".1",
			AccessMode: collection.AccessModeRead,
		})
	}
	frames, err := buildFrames(variables, 960)
	require.NoError(t, err)
	require.Len(t, frames, 1)
	assert.Len(t, frames[0].Variables, 30)
}

func s7Response(payload []byte, dataLength int) []byte {
	response := make([]byte, 21+len(payload))
	binutil.WriteUint16BigEndian(response[15:], uint16(dataLength))
	copy(response[21:], payload)
	return response
}

func TestValidateResponseItems(t *testing.T) {
	frame := &s7Frame{
		DataLength: 14,
		Items: []*frameItem{
			{Length: 2},
			{Length: 4},
		},
	}
	payload := []byte{
		0xFF, 0x02, 0x00, 0x10, 0x00, 0x2A,
		0xFF, 0x02, 0x00, 0x20, 0x3F, 0x80, 0x00, 0x00,
	}
	_, err := validateResponse(s7Response(payload, 6+8), frame)
	require.NoError(t, err)

	cases := [][]byte{
		{0xFE, 0x02, 0x00, 0x10, 0x00, 0x2A, 0xFF, 0x02, 0x00, 0x20, 0x3F, 0x80, 0x00, 0x00},
		{0xFF, 0x01, 0x00, 0x10, 0x00, 0x2A, 0xFF, 0x02, 0x00, 0x20, 0x3F, 0x80, 0x00, 0x00},
		{0xFF, 0x02, 0x00, 0x08, 0x00, 0x2A, 0xFF, 0x02, 0x00, 0x20, 0x3F, 0x80, 0x00, 0x00},
		{0xFF, 0x02, 0x00, 0x10, 0x00},
	}
	for _, itemPayload := range cases {
		_, err := validateResponse(s7Response(itemPayload, 6), frame)
		require.Error(t, err)
	}
}

func TestS7ClientAvailableAfterClose(t *testing.T) {
	left, right := net.Pipe()
	defer right.Close()
	client := &s7Client{conn: left}
	assert.True(t, client.Available())
	require.NoError(t, client.Close())
	assert.False(t, client.Available())
}

func TestBitDataType(t *testing.T) {
	length, err := variableByteLength(&collection.Variable{Name: "bit", DataType: "BIT"}, parsedAddress{area: areaI, address: 0})
	require.NoError(t, err)
	assert.Equal(t, uint16(1), length)

	_, err = variableByteLength(&collection.Variable{Name: "bit", DataType: "BIT"}, parsedAddress{area: areaDB, blockSize: 1, address: 0, length: 10})
	require.Error(t, err)

	frame := &s7Frame{Variables: []*frameVariable{
		{Variable: &collection.Variable{Name: "bit", DataType: "BIT"}, Start: 4, Bit: 2},
		{Variable: &collection.Variable{Name: "flag", DataType: "BOOL"}, Start: 4, Bit: 2},
	}}
	values, err := decodeFrame(frame, []byte{0, 0, 0, 0, 0x04})
	require.NoError(t, err)
	require.Len(t, values, 2)
	assert.Equal(t, int(1), values[0].value)
	assert.Equal(t, true, values[1].value)

	request, err := buildWriteRequest(&collection.Variable{Name: "bit", DataType: "BIT", Address: "DB1.DBX0.2"}, 1)
	require.NoError(t, err)
	assert.Equal(t, uint8(0x01), request[22])
	assert.Equal(t, []byte{0x00, 0x03, 0x00, 0x01, 0x01}, request[len(request)-5:])
}

func TestParseVAddressRejectsUnsupportedForms(t *testing.T) {
	for _, value := range []string{"V0", "VW0.10", "VD0.4", "VX0.0", "VW"} {
		_, err := parseAddress(value)
		require.Error(t, err, value)
	}
}

func TestS7200Config(t *testing.T) {
	created, err := NewS7200Collection(&CreateReq{
		Name:             "s7200-default",
		CollectorCycleMs: 1000,
		ProtocolConfig:   ProtocolConfig{Location: "127.0.0.1"},
	})
	require.NoError(t, err)
	assert.Equal(t, DriverModelS7200, created.GetDriverModel())
	assert.Equal(t, uint16(DefaultLocalTSAP), created.LocalTSAP)
	assert.Equal(t, uint16(DefaultRemoteTSAP), created.RemoteTSAP)

	config, err := created.ConfigJSON()
	require.NoError(t, err)
	assert.Contains(t, string(config), `"localTsap":"0x1000"`)
	assert.Contains(t, string(config), `"remoteTsap":"0x1000"`)
	assert.NotContains(t, string(config), `"rack"`)
	assert.NotContains(t, string(config), `"connectionType"`)

	loaded := &S7Collection{}
	loaded.SetDriverModel(DriverModelS7200)
	require.NoError(t, loaded.ApplyConfig(config))
	assert.Equal(t, created.Location, loaded.Location)
	assert.Equal(t, created.LocalTSAP, loaded.LocalTSAP)
	assert.Equal(t, created.RemoteTSAP, loaded.RemoteTSAP)
}

func TestFlexibleTSAPRejectsNonHex(t *testing.T) {
	for _, value := range []string{`{"localTsap":"4096"}`, `{"localTsap":"0xGGGG"}`, `{"localTsap":4096}`} {
		var config ProtocolConfig
		require.Error(t, json.Unmarshal([]byte(value), &config), value)
	}
}

func TestS7200HandlerAndFactory(t *testing.T) {
	handler := NewS7200CreateHandler()
	assert.Equal(t, DriverType, handler.DriverType())
	assert.Equal(t, DriverModelS7200, handler.DriverModel())
	assert.Equal(t, DriverModelS7200, NewS7200CollectorFactory().DriverModel())

	created, config, err := handler.BuildCreate(&CreateReq{
		Name:             "s7200-handler",
		CollectorCycleMs: 1000,
		ProtocolConfig:   ProtocolConfig{Location: "127.0.0.1"},
	})
	require.NoError(t, err)
	assert.Equal(t, DriverModelS7200, created.GetCollectionMeta().DriverModel)
	assert.Contains(t, string(config), `"localTsap":"0x1000"`)
}

func TestBuildFramesMapsVAreaToDB1(t *testing.T) {
	frames, err := buildFrames([]*collection.Variable{{
		Name: "v", DataType: "INT16", Address: "VW0", AccessMode: collection.AccessModeRead,
	}}, 480)
	require.NoError(t, err)
	require.Len(t, frames, 1)
	require.Len(t, frames[0].Items, 1)
	assert.Equal(t, []byte{0x12, 0x0a, 0x10, 0x02, 0x00, 0x02, 0x00, 0x01, 0x84, 0x00, 0x00, 0x00}, frames[0].Items[0].RequestData)
}
