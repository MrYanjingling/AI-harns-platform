package iec61850

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	iec "github.com/otfabric/go-iec61850"
	"github.com/otfabric/go-mms"
)

type client struct {
	api *iec.Client
}

func openClient(ctx context.Context, host string, port int, timeout time.Duration) (*client, error) {
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("invalid IEC 61850 port")
	}
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	api, err := iec.Dial(dialCtx, fmt.Sprintf("%s:%d", host, port), iec.DialOptions{
		MMS: mms.DialOptions{RequestTimeout: timeout},
	})
	if err != nil {
		return nil, err
	}
	return &client{api: api}, nil
}

func (c *client) Available() bool {
	if c == nil || c.api == nil || c.api.MMS() == nil {
		return false
	}
	select {
	case <-c.api.MMS().Done():
		return false
	default:
		return true
	}
}

func (c *client) Close() error {
	if c == nil || c.api == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return c.api.Close(ctx)
}

func (c *client) Read(ctx context.Context, address Address) (any, error) {
	if c == nil || c.api == nil {
		return nil, fmt.Errorf("IEC 61850 client is not connected")
	}
	value, err := c.api.Read(ctx, address.Reference)
	if err != nil {
		return nil, err
	}
	return convertValue(value)
}

func (c *client) Write(ctx context.Context, address Address, value any, dataType string) error {
	if c == nil || c.api == nil {
		return fmt.Errorf("IEC 61850 client is not connected")
	}
	mmsValue, err := encodeValue(value, dataType)
	if err != nil {
		return err
	}
	return c.api.Write(ctx, address.Reference, mmsValue)
}

func convertValue(value *iec.Value) (any, error) {
	if value == nil {
		return nil, fmt.Errorf("IEC 61850 value is nil")
	}
	switch value.Type() {
	case mms.ValueTypeBoolean:
		return value.Bool()
	case mms.ValueTypeInteger:
		return value.Int64()
	case mms.ValueTypeUnsigned:
		return value.Uint64()
	case mms.ValueTypeFloat:
		return value.Float64()
	case mms.ValueTypeVisibleString:
		return value.VisibleString()
	case mms.ValueTypeMmsString:
		return value.MmsString()
	case mms.ValueTypeOctetString:
		data, err := value.OctetString()
		if err != nil {
			return nil, err
		}
		return fmt.Sprintf("%X", data), nil
	case mms.ValueTypeBitString:
		bits, err := value.BitString()
		if err != nil {
			return nil, err
		}
		bitLength, ok := value.MMS().BitStringLength()
		if !ok {
			return nil, fmt.Errorf("invalid IEC 61850 bit string")
		}
		if bitLength == 1 {
			return len(bits) > 0 && bits[0]&0x80 != 0, nil
		}
		return fmt.Sprintf("%X", bits), nil
	case mms.ValueTypeUTCTime:
		decoded, ok := value.MMS().UTCTime()
		if !ok {
			return nil, fmt.Errorf("invalid IEC 61850 UTC time")
		}
		return decoded, nil
	case mms.ValueTypeReal:
		decoded, ok := value.MMS().Real()
		if !ok {
			return nil, fmt.Errorf("invalid IEC 61850 REAL")
		}
		return decoded, nil
	case mms.ValueTypeBCD:
		decoded, ok := value.MMS().BCD()
		if !ok {
			return nil, fmt.Errorf("invalid IEC 61850 BCD")
		}
		return decoded, nil
	case mms.ValueTypeObjectIdentifier:
		decoded, ok := value.MMS().ObjectIdentifier()
		if !ok {
			return nil, fmt.Errorf("invalid IEC 61850 object identifier")
		}
		return decoded, nil
	case mms.ValueTypeStructure, mms.ValueTypeArray, mms.ValueTypeBooleanArray:
		return value.String(), nil
	case mms.ValueTypeDataAccessError:
		return nil, fmt.Errorf("IEC 61850 data access error: %s", value.String())
	default:
		return value.String(), nil
	}
}

func encodeValue(value any, dataType string) (*mms.Value, error) {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "bool", "boolean", "bit":
		boolean, ok := value.(bool)
		if !ok {
			number, numberOK := numericUint(value)
			if !numberOK {
				return nil, fmt.Errorf("IEC 61850 bool value is invalid")
			}
			boolean = number != 0
		}
		return mms.NewBoolean(boolean), nil
	case "int8", "int16", "int32", "int64", "int":
		number, ok := numericInt(value)
		if !ok {
			return nil, fmt.Errorf("IEC 61850 integer value is invalid")
		}
		return mms.NewInteger(number), nil
	case "uint8", "uint16", "uint32", "uint64", "uint":
		number, ok := numericUint(value)
		if !ok {
			return nil, fmt.Errorf("IEC 61850 unsigned value is invalid")
		}
		return mms.NewUnsigned(number), nil
	case "float", "float32", "float64", "double":
		number, ok := numericFloat(value)
		if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
			return nil, fmt.Errorf("IEC 61850 float value is invalid")
		}
		return mms.NewFloat(number), nil
	case "string":
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("IEC 61850 string value is invalid")
		}
		return mms.NewVisibleString(text), nil
	default:
		return nil, fmt.Errorf("unsupported IEC 61850 data type %q", dataType)
	}
}

func numericInt(value any) (int64, bool) {
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
	default:
		return 0, false
	}
}
func numericUint(value any) (uint64, bool) {
	switch number := value.(type) {
	case int:
		return uint64(number), number >= 0
	case uint8:
		return uint64(number), true
	case uint16:
		return uint64(number), true
	case uint32:
		return uint64(number), true
	case uint64:
		return number, true
	default:
		return 0, false
	}
}
func numericFloat(value any) (float64, bool) {
	switch number := value.(type) {
	case float32:
		return float64(number), true
	case float64:
		return number, true
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	case uint64:
		return float64(number), true
	default:
		return 0, false
	}
}

func isTransportError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, iec.ErrClosed) || errors.Is(err, mms.ErrClosed) || errors.Is(err, mms.ErrInvokeTimeout) ||
		errors.Is(err, mms.ErrServerConnClosed) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "timeout") || strings.Contains(message, "connection") || strings.Contains(message, "closed") ||
		strings.Contains(message, "refused") || strings.Contains(message, "unreachable") || strings.Contains(message, "eof")
}
