package codesys3

import (
	"encoding/binary"
	"fmt"
)

type tag struct {
	ID       int
	Data     []byte
	Children []*tag
}

func encodeVarint(value int) []byte {
	out := make([]byte, 0, 5)
	for {
		item := byte(value & 0x7f)
		value >>= 7
		if value != 0 {
			item |= 0x80
		}
		out = append(out, item)
		if value == 0 {
			return out
		}
	}
}

func decodeVarint(data []byte, offset int) (int, int, error) {
	value := 0
	shift := 0
	for {
		if offset >= len(data) {
			return 0, offset, fmt.Errorf("truncated CODESYS varint")
		}
		item := data[offset]
		offset++
		value |= int(item&0x7f) << shift
		if item&0x80 == 0 {
			return value, offset, nil
		}
		shift += 7
		if shift > 35 {
			return 0, offset, fmt.Errorf("CODESYS varint is too long")
		}
	}
}

func encodeTag(item *tag) []byte {
	var body []byte
	if item.ID&0x80 != 0 {
		for _, child := range item.Children {
			body = append(body, encodeTag(child)...)
		}
	} else {
		body = item.Data
	}
	out := encodeVarint(item.ID)
	sizeBytes := encodeVarint(len(body))
	out = append(out, sizeBytes...)
	// CODESYS V3 aligns tag headers (ID + size varint) to 4-byte boundaries using
	// LEB128 continuation bytes (0x80...0x00) without changing the decoded length.
	if rem := len(out) % 4; rem != 0 {
		pad := 4 - rem
		out[len(out)-1] |= 0x80
		for i := 0; i < pad-1; i++ {
			out = append(out, 0x80)
		}
		out = append(out, 0x00)
	}
	return append(out, body...)
}

func encodeTags(items []*tag) []byte {
	out := make([]byte, 0)
	for _, item := range items {
		out = append(out, encodeTag(item)...)
	}
	return out
}

func parseTags(data []byte) ([]*tag, error) {
	items := make([]*tag, 0)
	offset := 0
	for offset < len(data) {
		id, next, err := decodeVarint(data, offset)
		if err != nil {
			return nil, err
		}
		length, next, err := decodeVarint(data, next)
		if err != nil {
			return nil, err
		}
		if next+length > len(data) {
			return nil, fmt.Errorf("CODESYS tag overruns payload")
		}
		item := &tag{ID: id}
		if id&0x80 != 0 {
			item.Children, err = parseTags(data[next : next+length])
			if err != nil {
				return nil, err
			}
		} else {
			item.Data = append([]byte{}, data[next:next+length]...)
		}
		items = append(items, item)
		offset = next + length
	}
	return items, nil
}

func findTag(items []*tag, id int) *tag {
	for _, item := range items {
		if item.ID == id {
			return item
		}
		for _, child := range item.Children {
			if found := findTag([]*tag{child}, id); found != nil {
				return found
			}
		}
	}
	return nil
}

func u32Tag(id int, value uint32) *tag {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, value)
	return &tag{ID: id, Data: data}
}

func u16Tag(id int, value uint16) *tag {
	data := make([]byte, 2)
	binary.LittleEndian.PutUint16(data, value)
	return &tag{ID: id, Data: data}
}

func cstrTag(id int, value string) *tag {
	return &tag{ID: id, Data: append([]byte(value), 0)}
}

func containerTag(id int, children ...*tag) *tag {
	return &tag{ID: id, Children: children}
}

func tagU32(item *tag) (uint32, error) {
	if item == nil || len(item.Data) < 4 {
		return 0, fmt.Errorf("CODESYS tag has no u32 payload")
	}
	return binary.LittleEndian.Uint32(item.Data), nil
}
