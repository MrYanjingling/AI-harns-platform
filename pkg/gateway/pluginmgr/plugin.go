// Package pluginmgr defines the shared Gateway driver plugin ABI.
package pluginmgr

import (
	"context"
	"encoding/json"
	"errors"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/connpool"
)

const (
	ABIVersion       = "v1"
	DriverSymbolName = "GatewayDriver"
)

var (
	ErrInvalidConfig = errors.New("invalid plugin config")
	ErrConnect       = errors.New("plugin connect failed")
	ErrRead          = errors.New("plugin read failed")
	ErrDecode        = errors.New("plugin decode failed")
	ErrWrite         = errors.New("plugin write failed")
)

type TransportKind string

const (
	TransportTCP    TransportKind = "tcp"
	TransportUDP    TransportKind = "udp"
	TransportSerial TransportKind = "serial"
	TransportCustom TransportKind = "custom"
)

// Descriptor identifies one plugin build.
type Descriptor struct {
	DriverType  string `json:"driverType"`
	DriverModel string `json:"driverModel"`
	PluginName  string `json:"pluginName"`
	Version     string `json:"version"`
	ABIVersion  string `json:"abiVersion"`
	GOOS        string `json:"goos"`
	GOARCH      string `json:"goarch"`
	GoVersion   string `json:"goVersion"`
}

// Transport is the plugin transport ABI shared with built-in collectors.
type Transport = connpool.Transport

// ReadFrame is one logical read request returned by a plugin.
type ReadFrame struct {
	Request   []byte
	Variables []*collection.Variable
	Context   any
}

// DecodedValue is one decoded variable value.
type DecodedValue struct {
	Variable *collection.Variable
	Value    any
}

// Driver is the narrow protocol ABI implemented by custom driver plugins.
// The host owns polling, connection pooling, status transitions and calculations.
type Driver interface {
	Descriptor() Descriptor
	ValidateConfig(config json.RawMessage) error
	TransportKind(config json.RawMessage) (TransportKind, error)
	NewTransport(ctx context.Context, config json.RawMessage) (Transport, error)

	BuildReadFrames(variables []*collection.Variable, config json.RawMessage) ([]ReadFrame, error)
	DecodeRead(frame ReadFrame, response []byte) ([]DecodedValue, error)

	BuildWrite(action collection.ActionValue, config json.RawMessage) ([]byte, error)
	DecodeWrite(response []byte) error
}

// DriverFactory is exported by plugins as GatewayDriver.
type DriverFactory func() Driver
