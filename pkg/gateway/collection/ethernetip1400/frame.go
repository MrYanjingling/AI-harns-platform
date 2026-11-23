package ethernetip1400

import (
	"fmt"

	"harns-platform/pkg/gateway/collection"
)

type pcccFrameVariable struct {
	Variable *collection.Variable
	Address  pcccAddress
}

type pcccFrame struct {
	Request   []byte
	Variables []*pcccFrameVariable
}

type decodedValue struct {
	Variable *collection.Variable
	Value    any
}

func buildReadFrames(variables []*collection.Variable) ([]*pcccFrame, error) {
	frames := make([]*pcccFrame, 0, len(variables))
	for _, variable := range variables {
		if variable == nil || variable.IsCalculated || variable.AccessMode == collection.AccessModeWrite {
			continue
		}
		address, err := parseAddress(variable.Address, variable.DataType)
		if err != nil {
			return nil, fmt.Errorf("variable %s: %w", variable.Name, err)
		}
		request, err := readRequest(address, variable.DataType)
		if err != nil {
			return nil, fmt.Errorf("variable %s: %w", variable.Name, err)
		}
		frames = append(frames, &pcccFrame{
			Request:   request,
			Variables: []*pcccFrameVariable{{Variable: variable, Address: address}},
		})
	}
	return frames, nil
}

func decodeFrame(frame *pcccFrame, response []byte) ([]decodedValue, error) {
	if frame == nil || len(frame.Variables) != 1 {
		return nil, fmt.Errorf("invalid ethernet/ip 1400 frame")
	}
	item := frame.Variables[0]
	value, err := parseReadResponse(response, item.Address, item.Variable.DataType)
	if err != nil {
		return nil, fmt.Errorf("variable %s: %w", item.Variable.Name, err)
	}
	return []decodedValue{{
		Variable: item.Variable,
		Value:    collection.ApplyTransform(value, item.Variable),
	}}, nil
}
