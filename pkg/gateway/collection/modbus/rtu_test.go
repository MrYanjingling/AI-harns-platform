package modbus

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/utils/binutil"
	"harns-platform/pkg/utils/crcutil"
)

func rtuFrame(parts ...byte) []byte {
	frame := append([]byte(nil), parts...)
	crc := make([]byte, 2)
	binutil.WriteUint16BigEndian(crc, crcutil.CheckCrc16sum(frame))
	return append(frame, crc...)
}

func TestReadRTUReadResponseReadsNormalFrame(t *testing.T) {
	frame := rtuFrame(0x01, 0x03, 0x02, 0x00, 0x0A)
	response, err := ReadRTUReadResponse(bytes.NewReader(frame))
	require.NoError(t, err)
	assert.Equal(t, frame, response)
}

func TestReadRTUReadResponseReadsExceptionFrame(t *testing.T) {
	frame := rtuFrame(0x01, 0x83, 0x02)
	response, err := ReadRTUReadResponse(bytes.NewReader(frame))
	require.NoError(t, err)
	assert.Equal(t, frame, response)

	_, err = ValidateRTUResponse(response, &Frame{Slave: 1, Function: FunctionReadHoldingRegisters})
	require.Error(t, err)
	assert.True(t, IsException(err))
	assert.Equal(t, "modbus exception: illegal data address (function 3, code 2)", err.Error())
}

func TestReadRTUWriteResponseReadsExceptionFrame(t *testing.T) {
	frame := rtuFrame(0x01, 0x86, 0x02)
	response, err := ReadRTUWriteResponse(bytes.NewReader(frame))
	require.NoError(t, err)
	assert.Equal(t, frame, response)

	err = ValidateRTUWriteResponse(response, 1, 6)
	require.Error(t, err)
	assert.True(t, IsException(err))
	assert.Equal(t, "modbus exception: illegal data address (function 6, code 2)", err.Error())
}
