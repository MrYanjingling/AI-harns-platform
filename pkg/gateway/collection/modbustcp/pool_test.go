package modbustcp

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExchangeConnReadsExceptionResponse(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	request := make([]byte, 12)
	go func() {
		input := make([]byte, len(request))
		if _, err := io.ReadFull(server, input); err != nil {
			return
		}
		_, _ = server.Write([]byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x03, 0x01, 0x84, 0x02})
	}()

	started := time.Now()
	response, err := exchangeConn(client, request, time.Second)
	require.NoError(t, err)
	assert.Len(t, response, 9)
	assert.Less(t, time.Since(started), 500*time.Millisecond)
}

func TestExchangeConnRejectsInvalidFrameLength(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	request := make([]byte, 12)
	go func() {
		input := make([]byte, len(request))
		if _, err := io.ReadFull(server, input); err != nil {
			return
		}
		_, _ = server.Write([]byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x01})
	}()

	_, err := exchangeConn(client, request, time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "frame length")
}
