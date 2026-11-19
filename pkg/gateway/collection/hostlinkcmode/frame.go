package hostlinkcmode

import (
	"fmt"
	"sort"

	"harns-platform/pkg/gateway/collection"
)

type hostlinkFrameVariable struct {
	Variable *collection.Variable
	Address  parsedAddress
	Offset   uint16
}

type hostlinkFrame struct {
	Address   parsedAddress
	Request   []byte
	Variables []*hostlinkFrameVariable
}

type decodedValue struct {
	Variable *collection.Variable
	Value    any
}

func buildReadFrames(variables []*collection.Variable, maxReadWords int) ([]*hostlinkFrame, error) {
	if maxReadWords <= 0 {
		maxReadWords = DefaultMaxReadWords
	}
	tags := make([]*hostlinkFrameVariable, 0, len(variables))
	for _, variable := range variables {
		if variable == nil || variable.IsCalculated || variable.AccessMode == collection.AccessModeWrite {
			continue
		}
		address, err := parseAddress(variable.Address, variable.DataType)
		if err != nil {
			return nil, fmt.Errorf("variable %s: %w", variable.Name, err)
		}
		if int(address.length) > maxReadWords {
			return nil, fmt.Errorf("variable %s length %d exceeds maxReadWords %d", variable.Name, address.length, maxReadWords)
		}
		tags = append(tags, &hostlinkFrameVariable{Variable: variable, Address: address})
	}
	sort.SliceStable(tags, func(i, j int) bool {
		left, right := tags[i].Address, tags[j].Address
		if left.unit != right.unit {
			return left.unit < right.unit
		}
		if left.area != right.area {
			return left.area < right.area
		}
		if left.isBit != right.isBit {
			return !left.isBit
		}
		if left.address != right.address {
			return left.address < right.address
		}
		return left.length < right.length
	})
	frames := make([]*hostlinkFrame, 0)
	for _, tag := range tags {
		address := tag.Address
		if len(frames) > 0 {
			last := frames[len(frames)-1]
			lastEnd := uint32(last.Address.address) + uint32(last.Address.length)
			newEnd := uint32(address.address) + uint32(address.length)
			if last.Address.unit == address.unit &&
				last.Address.area == address.area &&
				last.Address.isBit == address.isBit &&
				uint32(address.address) <= lastEnd &&
				newEnd-uint32(last.Address.address) <= uint32(maxReadWords) {
				tag.Offset = uint16(uint32(address.address) - uint32(last.Address.address))
				last.Variables = append(last.Variables, tag)
				if newEnd > lastEnd {
					last.Address.length = uint16(newEnd - uint32(last.Address.address))
					last.Request = buildReadRequest(last.Address)
				}
				continue
			}
		}
		frameAddress := address
		frameAddress.length = address.length
		tag.Offset = 0
		frames = append(frames, &hostlinkFrame{
			Address:   frameAddress,
			Request:   buildReadRequest(frameAddress),
			Variables: []*hostlinkFrameVariable{tag},
		})
	}
	return frames, nil
}

func decodeFrame(frame *hostlinkFrame, response response) ([]decodedValue, error) {
	values := make([]decodedValue, 0, len(frame.Variables))
	var firstErr error
	if frame.Address.isBit {
		for _, item := range frame.Variables {
			value, err := decodeBitValue(response.data, int(item.Offset), item.Variable.DataType)
			if err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("variable %s: %w", item.Variable.Name, err)
				}
				continue
			}
			values = append(values, decodedValue{Variable: item.Variable, Value: collection.ApplyTransform(value, item.Variable)})
		}
		return values, firstErr
	}
	raw, err := decodeHexWords(response.data, int(frame.Address.length))
	if err != nil {
		return nil, err
	}
	for _, item := range frame.Variables {
		start := int(item.Offset) * 2
		length := int(item.Address.length) * 2
		if start+length > len(raw) {
			if firstErr == nil {
				firstErr = fmt.Errorf("variable %s response is too short", item.Variable.Name)
			}
			continue
		}
		value, err := decodeWords(raw[start:start+length], item.Address, item.Variable.DataType)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("variable %s: %w", item.Variable.Name, err)
			}
			continue
		}
		values = append(values, decodedValue{Variable: item.Variable, Value: collection.ApplyTransform(value, item.Variable)})
	}
	return values, firstErr
}
