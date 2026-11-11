package s7

import (
	"fmt"
	"strings"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/utils/binutil"
)

type frameVariable struct {
	Variable  *collection.Variable
	Start     uint
	Bit       uint8
	BlockSize uint
}

type frameItem struct {
	RequestData []byte
	Start       uint
}

type s7Frame struct {
	Area       storeArea
	Request    []byte
	DataLength int
	Variables  []*frameVariable
}

func buildFrames(variables []*collection.Variable, pdu uint16) ([]*s7Frame, error) {
	groups := make(map[storeArea][]*collection.Variable)
	for _, variable := range variables {
		if variable == nil || variable.IsCalculated || variable.AccessMode == collection.AccessModeWrite {
			continue
		}
		if strings.EqualFold(variable.DataType, "string") {
			return nil, fmt.Errorf("variable %s: s7 string is not supported", variable.Name)
		}
		parsed, err := parseAddress(variable.Address)
		if err != nil {
			return nil, fmt.Errorf("variable %s: %w", variable.Name, err)
		}
		groups[parsed.area] = append(groups[parsed.area], variable)
	}
	maxItems := (int(pdu) - 19) / 12
	if len(groups) > 0 && maxItems < 1 {
		return nil, fmt.Errorf("s7 pdu %d is too small", pdu)
	}
	frames := make([]*s7Frame, 0)
	for area, values := range groups {
		sortVariables(values)
		items := make([]*frameItem, 0)
		itemMap := make(map[string]*frameItem)
		frameVars := make([]*frameVariable, 0)
		offset := 0
		flush := func() {
			if len(items) == 0 {
				return
			}
			frames = append(frames, newS7Frame(area, frameVars, items, offset))
			items = nil
			itemMap = make(map[string]*frameItem)
			frameVars = nil
			offset = 0
		}
		for _, variable := range values {
			parsed, err := parseAddress(variable.Address)
			if err != nil {
				return nil, fmt.Errorf("variable %s: %w", variable.Name, err)
			}
			key := fmt.Sprintf("%d.%d.%d", parsed.area, parsed.blockSize, parsed.address)
			item, exists := itemMap[key]
			if !exists {
				item = &frameItem{
					Start: uint(offset),
					RequestData: newReadParameterItem(
						parsed.area.transportSize(),
						dataRequestLength(parsed.area, variable.DataType),
						uint16(parsed.blockSize),
						parsed.area.code(),
						parsed.address,
						parsed.bit,
					),
				}
				items = append(items, item)
				itemMap[key] = item
				offset += int(4 + dataResponseLength(parsed.area, variable.DataType))
			}
			frameVars = append(frameVars, &frameVariable{
				Variable:  variable,
				Start:     item.Start,
				Bit:       parsed.bit,
				BlockSize: parsed.blockSize,
			})
			if len(items) == maxItems {
				flush()
			}
		}
		flush()
	}
	return frames, nil
}

func newS7Frame(area storeArea, variables []*frameVariable, items []*frameItem, dataLength int) *s7Frame {
	data := []byte{0x03, 0x00, 0x00}
	maxBytes := 7 + 10 + 2 + len(items)*12
	data = append(data, uint8(maxBytes))
	data = append(data, 0x02, 0xf0, 0x80)
	data = append(data, 0x32, 0x01, 0x00, 0x00, 0x00, 0x01)
	data = append(data, binutil.Uint16ToBytesBigEndian(uint16(2+len(items)*12))...)
	data = append(data, binutil.Uint16ToBytesBigEndian(uint16(0))...)
	data = append(data, uint8(4), uint8(len(items)))
	for _, item := range items {
		data = append(data, item.RequestData...)
	}
	return &s7Frame{Area: area, Request: data, DataLength: dataLength, Variables: variables}
}

func newReadParameterItem(transportSize uint8, length uint16, dbNumber uint16, zone uint8, address uint32, bitAddress uint8) []byte {
	item := []byte{0x12, 0x0a, 0x10}
	item = append(item, transportSize)
	item = append(item, binutil.Uint16ToBytesBigEndian(length)...)
	item = append(item, binutil.Uint16ToBytesBigEndian(dbNumber)...)
	item = append(item, zone)
	item = append(item, uint8((address<<3)/256/256))
	item = append(item, uint8((address<<3)/256%256))
	item = append(item, uint8((address<<3)%256)+bitAddress)
	return item
}
