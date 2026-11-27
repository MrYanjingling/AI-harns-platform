package s7

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/connpool"
)

func TestBuildWriteRequest(t *testing.T) {
	request, err := buildWriteRequest(&collection.Variable{
		Name: "speed", DataType: "int16", Address: "DB1.DBW0", AccessMode: collection.AccessModeReadWrite,
	}, int16(42))
	require.NoError(t, err)
	assert.NotEmpty(t, request)
}

func TestBuildWriteRequestBitAndBoolTransport(t *testing.T) {
	cases := []struct {
		name      string
		variable  *collection.Variable
		value     any
		dataBytes []byte
	}{
		{
			name:      "bit true",
			variable:  &collection.Variable{Name: "b", DataType: "BIT", Address: "DB1.DBX0.2"},
			value:     1,
			dataBytes: []byte{0x00, 0x03, 0x00, 0x01, 0x01},
		},
		{
			name:      "bit false",
			variable:  &collection.Variable{Name: "b", DataType: "BIT", Address: "DB1.DBX0.2"},
			value:     0,
			dataBytes: []byte{0x00, 0x03, 0x00, 0x01, 0x00},
		},
		{
			name:      "bool true",
			variable:  &collection.Variable{Name: "b", DataType: "BOOL", Address: "DB1.DBX0.2"},
			value:     true,
			dataBytes: []byte{0x00, 0x03, 0x00, 0x01, 0x01},
		},
		{
			name:      "bool false",
			variable:  &collection.Variable{Name: "b", DataType: "BOOL", Address: "DB1.DBX0.2"},
			value:     false,
			dataBytes: []byte{0x00, 0x03, 0x00, 0x01, 0x00},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request, err := buildWriteRequest(tc.variable, tc.value)
			require.NoError(t, err)
			require.Greater(t, len(request), 23)
			assert.Equal(t, uint8(0x01), request[22])
			assert.Equal(t, tc.dataBytes, request[len(request)-len(tc.dataBytes):])
		})
	}
}

func TestBuildWriteRequestNumericParameterTransport(t *testing.T) {
	request, err := buildWriteRequest(&collection.Variable{
		Name: "speed", DataType: "int16", Address: "DB1.DBW0",
	}, int16(42))
	require.NoError(t, err)
	require.Greater(t, len(request), 23)
	assert.Equal(t, uint8(0x02), request[22])
	assert.Equal(t, []byte{0x00, 0x04, 0x00, 0x10, 0x00, 0x2A}, request[len(request)-6:])
}

func TestBuildWriteBatchesBoolOddPadding(t *testing.T) {
	actions := []collection.ActionValue{
		{Index: 0, Variable: &collection.Variable{Name: "b0", DataType: "BOOL", Address: "DB1.DBX0.0"}, Value: true},
		{Index: 1, Variable: &collection.Variable{Name: "b1", DataType: "BOOL", Address: "DB1.DBX0.1"}, Value: false},
		{Index: 2, Variable: &collection.Variable{Name: "b2", DataType: "BOOL", Address: "DB1.DBX0.2"}, Value: true},
	}
	items := make([]writeItem, 0, len(actions))
	for _, action := range actions {
		item, err := buildWriteItem(action)
		require.NoError(t, err)
		items = append(items, item)
	}
	batches := buildWriteBatches(items, 960)
	require.Len(t, batches, 1)
	req := batches[0].request

	// TPKT(4) + COTP(3) + S7Header(10) + Param(2 + 3*12=38) + Data(6 + 6 + 5=17) = 72
	assert.Equal(t, 72, len(req))
	assert.Equal(t, uint16(72), binary.BigEndian.Uint16(req[2:4]))
	assert.Equal(t, uint16(38), binary.BigEndian.Uint16(req[13:15]))
	assert.Equal(t, uint16(17), binary.BigEndian.Uint16(req[15:17]))
	assert.Equal(t, uint8(0x05), req[17])
	assert.Equal(t, uint8(3), req[18])

	// Data section starts at 7 + 10 + 38 = 55
	expectedData := []byte{
		0x00, 0x03, 0x00, 0x01, 0x01, 0x00, // b0 (non-last odd -> 1 fill byte)
		0x00, 0x03, 0x00, 0x01, 0x00, 0x00, // b1 (non-last odd -> 1 fill byte)
		0x00, 0x03, 0x00, 0x01, 0x01, // b2 (last item -> no fill byte)
	}
	assert.Equal(t, expectedData, req[55:])
}

func TestBuildWriteBatchesSplitsByMaxItemsAndPDU(t *testing.T) {
	items := make([]writeItem, 0, 25)
	for i := 0; i < 25; i++ {
		item, err := buildWriteItem(collection.ActionValue{
			Index:    i,
			Variable: &collection.Variable{Name: "b", DataType: "BOOL", Address: "DB1.DBX0.0"},
			Value:    i%2 == 0,
		})
		require.NoError(t, err)
		items = append(items, item)
	}

	// PDU 960 splits by s7WriteMaxItems (20 + 5)
	batches960 := buildWriteBatches(items, 960)
	require.Len(t, batches960, 2)
	assert.Len(t, batches960[0].actions, 20)
	assert.Len(t, batches960[1].actions, 5)

	// PDU 240: budget = 240 - 12 = 228; each BOOL item costs 12 + 5 (+1 pad) = 18 (last=17) -> max 12 items (11*18+17=215 <= 228, 13th would be 233 > 228)
	batches240 := buildWriteBatches(items, 240)
	require.Len(t, batches240, 3)
	assert.Len(t, batches240[0].actions, 12)
	assert.Len(t, batches240[1].actions, 12)
	assert.Len(t, batches240[2].actions, 1)
}

func TestDeliverActionRetriesAfterStaleConnectionEOF(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	var connCount atomic.Int32
	var writeReqCount atomic.Int32
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}
			idx := connCount.Add(1)
			go func(c net.Conn, connIdx int32) {
				defer c.Close()
				// COTP CR (22 bytes) -> COTP CC (22 bytes)
				cotpReq := make([]byte, 22)
				if _, err := io.ReadFull(c, cotpReq); err != nil {
					return
				}
				cotpResp := make([]byte, 22)
				cotpResp[5] = 0xd0
				if _, err := c.Write(cotpResp); err != nil {
					return
				}
				// S7 Setup (25 bytes) -> Setup ACK (27 bytes, PDU=960)
				setupReq := make([]byte, 25)
				if _, err := io.ReadFull(c, setupReq); err != nil {
					return
				}
				setupResp := make([]byte, 27)
				setupResp[8] = 0x03
				binary.BigEndian.PutUint16(setupResp[25:27], 960)
				if _, err := c.Write(setupResp); err != nil {
					return
				}
				// First connection simulates peer closing before/during first write (EOF)
				if connIdx == 1 {
					return
				}
				// Second connection handles the retried multi-item write request
				hdr := make([]byte, 4)
				if _, err := io.ReadFull(c, hdr); err != nil {
					return
				}
				total := int(binary.BigEndian.Uint16(hdr[2:4]))
				body := make([]byte, total-4)
				if _, err := io.ReadFull(c, body); err != nil {
					return
				}
				writeReqCount.Add(1)
				itemCount := int(body[14]) // offset 18 in full frame -> 14 in body
				respLen := 21 + itemCount
				resp := make([]byte, respLen)
				resp[0] = 0x03
				binary.BigEndian.PutUint16(resp[2:4], uint16(respLen))
				resp[4], resp[5], resp[6] = 0x02, 0xf0, 0x80
				resp[7], resp[8] = 0x32, 0x03
				binary.BigEndian.PutUint16(resp[13:15], 2)
				binary.BigEndian.PutUint16(resp[15:17], uint16(itemCount))
				resp[19], resp[20] = 0x05, uint8(itemCount)
				for i := 0; i < itemCount; i++ {
					resp[21+i] = 0xFF
				}
				_, _ = c.Write(resp)
			}(conn, idx)
		}
	}()

	cotp := newCOTPConnectMessage(0, 1, 1)
	pool, err := connpool.NewPool(context.Background(), 1, func(ctx context.Context) (connpool.Client, error) {
		return newS7Client(ctx, listener.Addr().String(), cotp, 960)
	})
	require.NoError(t, err)
	defer pool.Close()

	v0 := &collection.Variable{Name: "b_000", DataType: "BOOL", Address: "DB1.DBX0.0", AccessMode: collection.AccessModeReadWrite}
	v1 := &collection.Variable{Name: "b_001", DataType: "BOOL", Address: "DB1.DBX0.1", AccessMode: collection.AccessModeReadWrite}
	col := &S7Collection{
		CollectionMeta: collection.CollectionMeta{
			Variables: []*collection.Variable{v0, v1},
		},
		runtime: &collectorRuntime{pool: pool, pdu: 960},
	}

	err = col.DeliverAction(context.Background(), []collection.ActionValue{
		{Index: 0, Variable: v0, Value: true},
		{Index: 1, Variable: v1, Value: false},
	})
	require.NoError(t, err)
	assert.Equal(t, int32(2), connCount.Load(), "should reconnect once after EOF")
	assert.Equal(t, int32(1), writeReqCount.Load(), "2 BOOLs should be merged into 1 S7 write frame")
}
