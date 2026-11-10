package collection

import "encoding/json"

// CreateRequest is a protocol-specific collection create request.
type CreateRequest interface {
	Validate() error
}

// CreateHandler decodes and builds a protocol-specific collection.
type CreateHandler interface {
	DriverType() string
	DriverModel() string
	DecodeCreate(body []byte) (CreateRequest, error)
	BuildCreate(req CreateRequest) (Collector, json.RawMessage, error)
}
