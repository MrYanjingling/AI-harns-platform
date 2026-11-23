package ethernetip5500

import (
	"fmt"

	"harns-platform/pkg/gateway/collection"
)

type cipFrameVariable struct {
	Variable *collection.Variable
	Address  parsedAddress
}

type cipFrame struct {
	Request   []byte
	Variables []*cipFrameVariable
}

type decodedValue struct {
	Variable *collection.Variable
	Value    any
}

func buildReadFrames(variables []*collection.Variable) ([]*cipFrame, error) {
	frames := make([]*cipFrame, 0, len(variables))
	for _, variable := range variables {
		if variable == nil || variable.IsCalculated || variable.AccessMode == collection.AccessModeWrite {
			continue
		}
		address, err := parseAddress(variable.Address, variable.DataType)
		if err != nil {
			return nil, fmt.Errorf("variable %s: %w", variable.Name, err)
		}
		request, err := readRequest(address)
		if err != nil {
			return nil, fmt.Errorf("variable %s: %w", variable.Name, err)
		}
		frames = append(frames, &cipFrame{
			Request:   request,
			Variables: []*cipFrameVariable{{Variable: variable, Address: address}},
		})
	}
	return frames, nil
}

func decodeFrame(frame *cipFrame, response []byte) ([]decodedValue, error) {
	if frame == nil || len(frame.Variables) != 1 {
		return nil, fmt.Errorf("invalid ethernet/ip 5500 frame")
	}
	item := frame.Variables[0]
	data, _, err := parseReadResponse(response)
	if err != nil {
		return nil, err
	}
	value, err := decodeValue(data, item.Variable.DataType, item.Address)
	if err != nil {
		return nil, fmt.Errorf("variable %s: %w", item.Variable.Name, err)
	}
	return []decodedValue{{
		Variable: item.Variable,
		Value:    collection.ApplyTransform(value, item.Variable),
	}}, nil
}
