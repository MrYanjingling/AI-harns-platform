package knd

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"

	"harns-platform/pkg/gateway/collection"
)

type kndClient struct {
	baseURL string
	client  *http.Client
	timeout time.Duration
	closed  bool
}

func openKNDClient(host string, port int, timeout time.Duration) (*kndClient, error) {
	if strings.TrimSpace(host) == "" || port < 1 || port > 65535 {
		return nil, fmt.Errorf("invalid KND endpoint")
	}
	transport := &http.Transport{
		Proxy:                 nil,
		MaxIdleConns:          2,
		MaxIdleConnsPerHost:   1,
		IdleConnTimeout:       30 * time.Second,
		ResponseHeaderTimeout: timeout,
	}
	return &kndClient{
		baseURL: fmt.Sprintf("http://%s:%d", host, port),
		client:  &http.Client{Transport: transport, Timeout: timeout},
		timeout: timeout,
	}, nil
}

func (c *kndClient) Available() bool { return c != nil && !c.closed }
func (c *kndClient) Close() error {
	if c == nil {
		return nil
	}
	c.closed = true
	if transport, ok := c.client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
	return nil
}

func (c *kndClient) Read(ctx context.Context, address Address, variable *collection.Variable) (any, error) {
	root, err := c.request(ctx, http.MethodGet, address.path(), nil)
	if err != nil {
		return nil, err
	}
	if address.Kind == kindPLC {
		return decodePLCResponse(root, address, variable.DataType)
	}
	value := extractValue(root, address)
	if value == nil {
		return nil, fmt.Errorf("KND response does not contain %q", address.Key)
	}
	return convertJSONValue(value, variable.DataType)
}

func (c *kndClient) Write(ctx context.Context, address Address, variable *collection.Variable, value any) error {
	switch {
	case address.Kind == kindPLC && address.Area == "R":
		data, err := encodePLCBytes(value, variable.DataType, address.Size)
		if err != nil {
			return err
		}
		body := make([]int, len(data))
		for index, item := range data {
			body[index] = int(item)
		}
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		_, err = c.request(ctx, http.MethodPut, address.path(), payload)
		return err
	case address.Kind == kindVars:
		payload, err := json.Marshal(map[string]any{"val": value})
		if err != nil {
			return err
		}
		_, err = c.request(ctx, http.MethodPut, address.path(), payload)
		return err
	default:
		return fmt.Errorf("KND address %q is not writable", address.String())
	}
}

func (c *kndClient) request(ctx context.Context, method, path string, body []byte) (any, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("KND client is not initialized")
	}
	encodedBody := body
	if len(body) > 0 {
		var err error
		encodedBody, err = simplifiedchinese.GBK.NewEncoder().Bytes(body)
		if err != nil {
			return nil, err
		}
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(encodedBody))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json;charset=gb2312")
	response, err := c.client.Do(request)
	if err != nil {
		return nil, collection.ConnectionLost(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, collection.ConnectionLost(err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("KND HTTP status %d", response.StatusCode)
	}
	utf8Data, err := simplifiedchinese.GBK.NewDecoder().Bytes(data)
	if err != nil {
		return nil, err
	}
	var root any
	if err := json.Unmarshal(utf8Data, &root); err != nil {
		return nil, fmt.Errorf("invalid KND JSON: %w", err)
	}
	if value, ok := root.(map[string]any)["error"]; ok {
		if number, ok := value.(float64); ok && number != 0 {
			return nil, fmt.Errorf("KND error %v", number)
		}
	}
	return root, nil
}

func (a Address) path() string {
	switch a.Kind {
	case kindPLC:
		return fmt.Sprintf("/v1/plc/vm/%s%d?size=%d", a.Area, a.Number, a.Size)
	case kindVars:
		return fmt.Sprintf("/v1/vars/%d", a.Index)
	case kindSystemInfo, kindSystemStatus:
		return "/v1/status"
	case kindAlarms:
		return "/v1/alarms/"
	case kindAbsolute:
		return "/v1/coors/absolute"
	case kindMachine:
		return "/v1/coors/machine"
	case kindRelative:
		return "/v1/coors/relative"
	case kindCycleTime:
		return "/v1/cycletime"
	case kindWorkCounts:
		return "/v1/workcounts/"
	case kindWorkCountGoals:
		return "/v1/workcountgoals/"
	case kindFeedOverride:
		return "/v1/overrides/feed"
	case kindJogOverride:
		return "/v1/overrides/jog"
	case kindRapidOverride:
		return "/v1/overrides/rapid"
	case kindHandleOverride:
		return "/v1/overrides/handle"
	case kindSpindleOverride:
		return "/v1/overrides/sp/"
	case kindSpSpeed:
		return "/v1/sp/speeds/"
	case kindFeedrate:
		return "/v1/feedback/F"
	case kindWorkCoors:
		return fmt.Sprintf("/v1/workcoors/g%d", a.Number)
	case kindWorkCoorsCur:
		return "/v1/workcoors/cur"
	case kindProgCur:
		return "/v1/progs/cur"
	case kindProgExecStatus:
		return "/v1/progs/exec-status"
	default:
		return "/v1/status"
	}
}

func (a Address) String() string {
	if a.Kind == kindPLC {
		return a.Area + strconv.Itoa(a.Number)
	}
	return a.Key
}

func extractValue(root any, address Address) any {
	if address.Key != "" {
		if value := lookupKey(root, address.Key); value != nil {
			return value
		}
	}
	switch address.Kind {
	case kindAbsolute, kindMachine, kindRelative, kindWorkCoors:
		if address.Axis >= 0 {
			if value := lookupAxis(root, address.Axis); value != nil {
				return value
			}
		}
	case kindVars:
		if object, ok := root.(map[string]any); ok {
			if value, ok := object["val"]; ok {
				return value
			}
		}
	case kindFeedOverride, kindJogOverride, kindRapidOverride, kindHandleOverride:
		if object, ok := root.(map[string]any); ok {
			if value, ok := object["ov"]; ok {
				return value
			}
			if value, ok := object["val"]; ok {
				return value
			}
		}
	case kindSpindleOverride, kindSpSpeed:
		if object, ok := root.(map[string]any); ok {
			if value, ok := object[strconv.Itoa(address.Index)]; ok {
				return value
			}
			if value, ok := object["val"]; ok {
				return value
			}
		}
	case kindFeedrate:
		if object, ok := root.(map[string]any); ok {
			if value, ok := object["feedrate"]; ok {
				return value
			}
			if value, ok := object["val"]; ok {
				return value
			}
		}
	case kindProgCur:
		if object, ok := root.(map[string]any); ok {
			if value, ok := object["number"]; ok {
				return value
			}
			if value, ok := object["val"]; ok {
				return value
			}
		}
	case kindWorkCoorsCur:
		if object, ok := root.(map[string]any); ok {
			for _, g := range []string{"G54", "G55", "G56", "G57", "G58", "G59"} {
				if _, ok := object[g]; ok {
					return g
				}
			}
			if value, ok := object["val"]; ok {
				return value
			}
		}
	}
	return root
}

func lookupKey(value any, key string) any {
	switch item := value.(type) {
	case map[string]any:
		if found, ok := item[key]; ok {
			return found
		}
		for _, child := range item {
			if found := lookupKey(child, key); found != nil {
				return found
			}
		}
	case []any:
		for _, child := range item {
			if found := lookupKey(child, key); found != nil {
				return found
			}
		}
	}
	return nil
}

func lookupAxis(value any, axis int) any {
	switch item := value.(type) {
	case []any:
		if axis < len(item) {
			return item[axis]
		}
	case map[string]any:
		names := []string{"X", "Y", "Z", "A", "B", "C"}
		if axis >= 0 && axis < len(names) {
			if found, ok := item[names[axis]]; ok {
				return found
			}
			if found, ok := item[strings.ToLower(names[axis])]; ok {
				return found
			}
		}
		if found, ok := item["val"]; ok {
			return found
		}
	}
	return nil
}

func decodePLCResponse(root any, address Address, dataType string) (any, error) {
	if address.Area == "T" || address.Area == "C" {
		value := lookupKey(root, address.Suffix)
		if value == nil {
			return nil, fmt.Errorf("KND %s response has no %s", address.Area, address.Suffix)
		}
		return convertJSONValue(value, dataType)
	}
	data, err := jsonBytes(root)
	if err != nil {
		return nil, err
	}
	return decodePLCBytes(data, address, dataType)
}

func jsonBytes(value any) ([]byte, error) {
	switch item := value.(type) {
	case []any:
		data := make([]byte, len(item))
		for index, value := range item {
			number, ok := value.(float64)
			if !ok {
				return nil, fmt.Errorf("KND byte array contains a non-number")
			}
			data[index] = byte(number)
		}
		return data, nil
	case map[string]any:
		if value, ok := item["val"]; ok {
			return jsonBytes(value)
		}
	}
	return nil, fmt.Errorf("KND response is not a byte array")
}

func decodePLCBytes(data []byte, address Address, dataType string) (any, error) {
	if len(data) < address.Size {
		return nil, fmt.Errorf("short KND PLC response")
	}
	switch strings.ToLower(dataType) {
	case "bit", "bool":
		bit := (data[0] >> address.Bit) & 1
		if strings.EqualFold(dataType, "bit") {
			return bit, nil
		}
		return bit != 0, nil
	case "int8":
		return int8(data[0]), nil
	case "uint8":
		return data[0], nil
	case "int16":
		return int16(binary.LittleEndian.Uint16(data)), nil
	case "uint16":
		return binary.LittleEndian.Uint16(data), nil
	case "int32":
		return int32(binary.LittleEndian.Uint32(data)), nil
	case "uint32":
		return binary.LittleEndian.Uint32(data), nil
	case "float":
		return math.Float32frombits(binary.LittleEndian.Uint32(data)), nil
	case "int64":
		return int64(binary.LittleEndian.Uint64(data)), nil
	case "uint64":
		return binary.LittleEndian.Uint64(data), nil
	case "double":
		return math.Float64frombits(binary.LittleEndian.Uint64(data)), nil
	case "string":
		return strings.TrimRight(string(data[:address.Size]), "\x00"), nil
	default:
		return nil, fmt.Errorf("unsupported KND data type %q", dataType)
	}
}

func convertJSONValue(value any, dataType string) (any, error) {
	switch strings.ToLower(dataType) {
	case "bit", "uint8", "int8":
		number, ok := value.(float64)
		if !ok || number < -128 || number > 255 {
			return nil, fmt.Errorf("invalid KND integer value")
		}
		if strings.EqualFold(dataType, "bit") {
			return int(number) & 1, nil
		}
		return int64(number), nil
	case "bool":
		boolean, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("invalid KND bool value")
		}
		return boolean, nil
	case "uint16", "uint32", "uint64":
		number, ok := value.(float64)
		if !ok || number < 0 {
			return nil, fmt.Errorf("invalid KND unsigned value")
		}
		return uint64(number), nil
	case "int16", "int32", "int64":
		number, ok := value.(float64)
		if !ok {
			return nil, fmt.Errorf("invalid KND integer value")
		}
		return int64(number), nil
	case "float", "double":
		number, ok := value.(float64)
		if !ok {
			return nil, fmt.Errorf("invalid KND float value")
		}
		return number, nil
	case "string":
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("invalid KND string value")
		}
		return text, nil
	default:
		return value, nil
	}
}

func encodePLCBytes(value any, dataType string, size int) ([]byte, error) {
	data := make([]byte, size)
	switch strings.ToLower(dataType) {
	case "bit", "bool", "int8", "uint8":
		number, ok := toInt64(value)
		if !ok {
			return nil, fmt.Errorf("invalid KND byte value")
		}
		data[0] = byte(number)
	case "int16", "uint16":
		number, ok := toInt64(value)
		if !ok {
			return nil, fmt.Errorf("invalid KND int16 value")
		}
		binary.LittleEndian.PutUint16(data, uint16(number))
	case "int32", "uint32":
		number, ok := toInt64(value)
		if !ok {
			return nil, fmt.Errorf("invalid KND int32 value")
		}
		binary.LittleEndian.PutUint32(data, uint32(number))
	case "int64", "uint64":
		number, ok := toInt64(value)
		if !ok {
			return nil, fmt.Errorf("invalid KND int64 value")
		}
		binary.LittleEndian.PutUint64(data, uint64(number))
	case "float":
		number, ok := toFloat64(value)
		if !ok {
			return nil, fmt.Errorf("invalid KND float value")
		}
		binary.LittleEndian.PutUint32(data, math.Float32bits(float32(number)))
	case "double":
		number, ok := toFloat64(value)
		if !ok {
			return nil, fmt.Errorf("invalid KND double value")
		}
		binary.LittleEndian.PutUint64(data, math.Float64bits(number))
	case "string":
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("invalid KND string value")
		}
		copy(data, text)
	default:
		return nil, fmt.Errorf("unsupported KND data type %q", dataType)
	}
	return data, nil
}

func toInt64(value any) (int64, bool) {
	switch number := value.(type) {
	case int:
		return int64(number), true
	case int8:
		return int64(number), true
	case int16:
		return int64(number), true
	case int32:
		return int64(number), true
	case int64:
		return number, true
	case uint8:
		return int64(number), true
	case uint16:
		return int64(number), true
	case uint32:
		return int64(number), true
	case uint64:
		return int64(number), true
	case float32:
		return int64(number), true
	case float64:
		return int64(number), true
	default:
		return 0, false
	}
}

func toFloat64(value any) (float64, bool) {
	switch number := value.(type) {
	case float32:
		return float64(number), true
	case float64:
		return number, true
	default:
		integer, ok := toInt64(value)
		return float64(integer), ok
	}
}
