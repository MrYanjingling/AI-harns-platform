package modbustcp

import (
	"encoding/json"
	"fmt"

	"harns-platform/pkg/gateway/collection"
)

// CreateHandler decodes and builds Modbus TCP collections.
type CreateHandler struct{}

// DriverType returns the protocol type.
func (h *CreateHandler) DriverType() string { return DriverType }

// DriverModel returns the protocol model.
func (h *CreateHandler) DriverModel() string { return DriverModel }

// DecodeCreate decodes a Modbus TCP create request.
func (h *CreateHandler) DecodeCreate(body []byte) (collection.CreateRequest, error) {
	var req CreateReq
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

// BuildCreate builds a Modbus TCP collection and protocol config.
func (h *CreateHandler) BuildCreate(raw collection.CreateRequest) (collection.Collector, json.RawMessage, error) {
	req, ok := raw.(*CreateReq)
	if !ok {
		return nil, nil, fmt.Errorf("invalid modbusTcp create request")
	}
	item, err := NewCollection(req)
	if err != nil {
		return nil, nil, err
	}
	config, err := item.ConfigJSON()
	if err != nil {
		return nil, nil, err
	}
	return item, json.RawMessage(config), nil
}

var _ collection.CreateHandler = (*CreateHandler)(nil)
