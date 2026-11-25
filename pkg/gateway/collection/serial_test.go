package collection

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeSerialPort(t *testing.T) {
	assert.Empty(t, NormalizeSerialPort(" "))
	assert.Equal(t, "/dev/ttyUSB0", NormalizeSerialPort("/dev/ttyUSB0"))
	assert.Equal(t, `\\.\COM10`, NormalizeSerialPort(`\\.\COM10`))
	assert.Equal(t, "COM3", NormalizeSerialPort("COM3"))
	assert.Equal(t, "com3", NormalizeSerialPort("com3"))

	want := "/dev/ttyUSB0"
	if runtime.GOOS == "windows" {
		want = "ttyUSB0"
	}
	assert.Equal(t, want, NormalizeSerialPort("ttyUSB0"))
	assert.Equal(t, want, NormalizeSerialPort(" ttyUSB0 "))
}
