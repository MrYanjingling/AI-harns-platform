package ethernetip

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
	Multiple  bool
}

type decodedValue struct {
	Variable *collection.Variable
	Value    any
}

func buildReadFrames(variables []*collection.Variable) ([]*cipFrame, error) {
	type pending struct {
		variable *cipFrameVariable
		request  []byte
	}
	tags := make([]pending, 0, len(variables))
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
		tags = append(tags, pending{
			variable: &cipFrameVariable{Variable: variable, Address: address},
			request:  request,
		})
	}
	frames := make([]*cipFrame, 0)
	current := make([]pending, 0)
	cost := 0
	flush := func() error {
		if len(current) == 0 {
			return nil
		}
		if len(current) == 1 {
			frames = append(frames, &cipFrame{
				Request:   current[0].request,
				Variables: []*cipFrameVariable{current[0].variable},
			})
		} else {
			requests := make([][]byte, 0, len(current))
			variables := make([]*cipFrameVariable, 0, len(current))
			for _, item := range current {
				requests = append(requests, item.request)
				variables = append(variables, item.variable)
			}
			request, err := buildMultipleRequest(requests)
			if err != nil {
				return err
			}
			frames = append(frames, &cipFrame{
				Request:   request,
				Variables: variables,
				Multiple:  true,
			})
		}
		current = nil
		cost = 0
		return nil
	}
	for _, tag := range tags {
		itemCost := len(tag.request) + 16
		if len(current) > 0 && (len(current) >= maxBatchTags || cost+itemCost > maxBatchBytes) {
			if err := flush(); err != nil {
				return nil, err
			}
		}
		current = append(current, tag)
		cost += itemCost
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return frames, nil
}

func decodeFrame(frame *cipFrame, response []byte) ([]decodedValue, error) {
	if !frame.Multiple {
		data, _, err := parseReadResponse(response)
		if err != nil {
			return nil, err
		}
		value, err := decodeValue(data, frame.Variables[0].Variable.DataType, frame.Variables[0].Address)
		if err != nil {
			return nil, fmt.Errorf("variable %s: %w", frame.Variables[0].Variable.Name, err)
		}
		return []decodedValue{{
			Variable: frame.Variables[0].Variable,
			Value:    collection.ApplyTransform(value, frame.Variables[0].Variable),
		}}, nil
	}
	items, err := parseMultipleResponse(response)
	if err != nil {
		return nil, err
	}
	if len(items) != len(frame.Variables) {
		return nil, fmt.Errorf("ethernet/ip multiple response count %d does not match %d variables", len(items), len(frame.Variables))
	}
	values := make([]decodedValue, 0, len(items))
	var firstErr error
	for index, item := range frame.Variables {
		data, _, err := parseReadResponse(items[index])
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("variable %s: %w", item.Variable.Name, err)
			}
			continue
		}
		value, err := decodeValue(data, item.Variable.DataType, item.Address)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("variable %s: %w", item.Variable.Name, err)
			}
			continue
		}
		values = append(values, decodedValue{
			Variable: item.Variable,
			Value:    collection.ApplyTransform(value, item.Variable),
		})
	}
	return values, firstErr
}
