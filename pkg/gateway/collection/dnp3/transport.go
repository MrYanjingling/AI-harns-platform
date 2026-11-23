package dnp3

import "fmt"

func wrapTransport(seq byte, app []byte) []byte {
	payload := make([]byte, 1+len(app))
	payload[0] = 0xC0 | (seq & 0x3F)
	copy(payload[1:], app)
	return payload
}

func unwrapTransport(payload []byte) (byte, []byte, error) {
	if len(payload) < 1 {
		return 0, nil, fmt.Errorf("DNP3 transport payload is empty")
	}
	return payload[0] & 0x3F, payload[1:], nil
}
