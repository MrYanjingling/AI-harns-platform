package profinet

import (
	"encoding/json"
	"fmt"

	"harns-platform/pkg/gateway/collection"
)

type CreateHandler struct{}

func NewCreateHandler() *CreateHandler {
	return &CreateHandler{}
}

func (h *CreateHandler) DriverType() string  { return DriverType }
func (h *CreateHandler) DriverModel() string { return DriverModel }

func (h *CreateHandler) DecodeCreate(body []byte) (collection.CreateRequest, error) {
	var req CreateReq
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func (h *CreateHandler) BuildCreate(raw collection.CreateRequest) (collection.Collector, json.RawMessage, error) {
	req, ok := raw.(*CreateReq)
	if !ok {
		return nil, nil, fmt.Errorf("invalid profinet create request")
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
