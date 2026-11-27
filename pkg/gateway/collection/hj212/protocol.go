package hj212

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/meterlink"
)

type hjClient struct {
	link    *meterlink.Client
	timeout time.Duration
}

func openHJClient(ctx context.Context, link, port int, location string, baud, data int, parity, stop string, timeout time.Duration) (*hjClient, error) {
	item, err := meterlink.Open(ctx, link, location, port, timeout, meterlink.SerialMode(baud, data, parity, stop))
	if err != nil {
		return nil, err
	}
	return &hjClient{link: meterlink.NewClient(item, timeout), timeout: timeout}, nil
}

func (c *hjClient) Available() bool { return c != nil && c.link != nil && c.link.Available() }
func (c *hjClient) Close() error {
	if c == nil || c.link == nil {
		return nil
	}
	return c.link.Close()
}

func (c *hjClient) Read(ctx context.Context, address Address, st, pw, mn string) (any, error) {
	_ = ctx
	request := buildRequest(address.Command, st, pw, mn)
	if err := c.write(request); err != nil {
		return nil, collection.ConnectionLost(err)
	}
	var values map[string]string
	for attempt := 0; attempt < 4; attempt++ {
		response, err := c.readFrame()
		if err != nil {
			return nil, collection.ConnectionLost(err)
		}
		if isAckFrame(response) {
			continue
		}
		values, err = parseResponse(response)
		if err != nil {
			return nil, err
		}
		break
	}
	value, ok := values[address.Key]
	if !ok {
		return nil, fmt.Errorf("HJ212 response has no key %q", address.Key)
	}
	if address.Float {
		number, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid HJ212 numeric value %q", value)
		}
		return number, nil
	}
	return value, nil
}

func isAckFrame(frame []byte) bool {
	text := string(frame)
	return strings.Contains(text, "CN=9011") || strings.Contains(text, "CN=9012")
}

func buildRequest(command int, st, pw, mn string) []byte {
	now := time.Now()
	qn := now.Format("20060102150405") + fmt.Sprintf("%03d", now.Nanosecond()/1e6)
	data := fmt.Sprintf("QN=%s;ST=%s;CN=%d;PW=%s;MN=%s;Flag=4;CP=&&&&", qn, st, command, pw, mn)
	header := fmt.Sprintf("##%04d", len(data))
	crc := crc16([]byte(data))
	return []byte(fmt.Sprintf("%s%s%04X\r\n", header, data, crc))
}

func (c *hjClient) write(data []byte) error {
	if err := c.link.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return err
	}
	_, err := c.link.Write(data)
	return err
}

func (c *hjClient) readFrame() ([]byte, error) {
	if err := c.link.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, err
	}
	header := make([]byte, 6)
	if _, err := io.ReadFull(c.link, header); err != nil {
		return nil, err
	}
	if string(header[:2]) != "##" {
		return nil, fmt.Errorf("invalid HJ212 response header")
	}
	length, err := strconv.Atoi(string(header[2:]))
	if err != nil || length < 0 || length > 9999 {
		return nil, fmt.Errorf("invalid HJ212 response length")
	}
	body := make([]byte, length+6)
	if _, err := io.ReadFull(c.link, body); err != nil {
		return nil, err
	}
	return append(header, body...), nil
}

func parseResponse(frame []byte) (map[string]string, error) {
	text := string(frame)
	if !strings.HasPrefix(text, "##") || len(text) < 12 {
		return nil, fmt.Errorf("invalid HJ212 response header")
	}
	length, err := strconv.Atoi(text[2:6])
	if err != nil || length < 0 || 6+length > len(text) {
		return nil, fmt.Errorf("invalid HJ212 response length")
	}
	data := text[6 : 6+length]
	cpStart := strings.Index(data, "&&")
	cpEnd := strings.LastIndex(data, "&&")
	if cpStart < 0 || cpEnd <= cpStart {
		return map[string]string{}, nil
	}
	values := map[string]string{}
	for _, item := range strings.FieldsFunc(data[cpStart+2:cpEnd], func(r rune) bool { return r == ',' || r == ';' }) {
		parts := strings.SplitN(strings.TrimSpace(item), "=", 2)
		if len(parts) == 2 && parts[0] != "" {
			values[strings.TrimSpace(parts[0])] = parts[1]
		}
	}
	return values, nil
}

func crc16(data []byte) uint16 {
	var crc uint16 = 0xFFFF
	for _, value := range data {
		crc = (crc >> 8) ^ uint16(value)
		for bit := 0; bit < 8; bit++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}
