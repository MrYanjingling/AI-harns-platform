package codesys3

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"strings"
)

const (
	blockMagic      = 0xE8170100
	routerMarker    = 0xC5
	routerFlag1     = 0x6B
	routerFlag2     = 0x40
	serviceChannel  = 0x40
	serviceAddress  = 0x03
	serviceName     = 0x04
	nsResolveAddr   = 0xC202
	l4Data          = 0x01
	l4Ack           = 0x02
	l4Keepalive     = 0x03
	l4OpenChannel   = 0xC3
	l4CloseChannel  = 0xC4
	l4FlagFirst     = 0x01
	l4FlagRequest   = 0x80
	protoID         = 0xCD55
	serviceReplyBit = 0x80
	sgDevice        = 0x01
	sgIECVarAccess  = 0x09
	tagSession      = 0x21
	tagCrypto       = 0x22
	tagChallenge    = 0x23
)

func crc32IEEE(data []byte) uint32 { return crc32.ChecksumIEEE(data) }

func blockFrame(payload []byte) []byte {
	total := 8 + len(payload)
	frame := make([]byte, total)
	binary.LittleEndian.PutUint32(frame[0:4], blockMagic)
	binary.LittleEndian.PutUint32(frame[4:8], uint32(total))
	copy(frame[8:], payload)
	return frame
}

func parseBlockFrame(frame []byte) ([]byte, error) {
	if len(frame) < 8 || binary.LittleEndian.Uint32(frame[:4]) != blockMagic {
		return nil, fmt.Errorf("invalid CODESYS block frame")
	}
	total := int(binary.LittleEndian.Uint32(frame[4:8]))
	if total < 8 || total > len(frame) {
		return nil, fmt.Errorf("invalid CODESYS block length")
	}
	return frame[8:total], nil
}

func nodeAddress(ip string, port int, extra int) []byte {
	octetStrings := strings.Split(ip, ".")
	out := make([]byte, 0, 8)
	out = binary.BigEndian.AppendUint16(out, uint16(port))
	if len(octetStrings) == 4 {
		for _, text := range octetStrings {
			value := 0
			for _, ch := range text {
				if ch < '0' || ch > '9' {
					value = 0
					break
				}
				value = value*10 + int(ch-'0')
			}
			out = append(out, byte(value))
		}
	} else {
		out = append(out, 0, 0, 0, 0)
	}
	if extra != 0 {
		out = binary.BigEndian.AppendUint16(out, uint16(extra))
	}
	return out
}

func buildRouterDatagram(payload, src, dst []byte, serviceID, msgID byte) []byte {
	addrLen := byte(len(dst)/2)<<4 | byte(len(src)/2)
	header := []byte{routerMarker, routerFlag1, routerFlag2, serviceID, msgID, addrLen}
	header = append(header, dst...)
	header = append(header, src...)
	for len(header)%4 != 0 {
		header = append(header, 0)
	}
	return append(header, payload...)
}

func parseRouterDatagram(data []byte) (serviceID byte, payload []byte, src []byte, err error) {
	if len(data) < 6 || data[0] != routerMarker {
		return 0, nil, nil, fmt.Errorf("invalid CODESYS router datagram")
	}
	serviceID = data[3]
	addrLen := data[5]
	dstLen := int(addrLen>>4) * 2
	srcLen := int(addrLen&0x0f) * 2
	headerLen := 6 + dstLen + srcLen
	if rem := headerLen % 4; rem != 0 {
		headerLen += 4 - rem
	}
	if headerLen > len(data) {
		return 0, nil, nil, fmt.Errorf("short CODESYS router address")
	}
	src = data[6+dstLen : 6+dstLen+srcLen]
	payload = data[headerLen:]
	return serviceID, payload, src, nil
}

func buildResolveRequest(msgID byte, src, dst []byte) []byte {
	body := make([]byte, 8)
	binary.LittleEndian.PutUint16(body[0:2], nsResolveAddr)
	binary.LittleEndian.PutUint16(body[2:4], 0x0103)
	binary.LittleEndian.PutUint32(body[4:8], 0x6D240000)
	return buildRouterDatagram(body, src, dst, serviceAddress, msgID)
}

func buildOpenChannelRequest(requestID, bufferSize int) []byte {
	body := make([]byte, 20)
	body[0] = l4OpenChannel
	binary.LittleEndian.PutUint16(body[2:4], 0x0101)
	binary.LittleEndian.PutUint32(body[8:12], uint32(requestID))
	binary.LittleEndian.PutUint32(body[12:16], uint32(bufferSize))
	binary.LittleEndian.PutUint32(body[16:20], 5)
	return applyMetaChecksum(body)
}

func parseOpenChannelResponse(data []byte) (channelID int, commBuffer int, result uint16, err error) {
	if len(data) < 20 {
		return 0, 0, 0, fmt.Errorf("short CODESYS open channel response")
	}
	result = binary.LittleEndian.Uint16(data[12:14])
	channelID = int(binary.LittleEndian.Uint16(data[14:16]))
	commBuffer = int(binary.LittleEndian.Uint32(data[16:20]))
	return channelID, commBuffer, result, nil
}

func applyMetaChecksum(packet []byte) []byte {
	out := append([]byte{}, packet...)
	if len(out) >= 8 {
		for i := 4; i < 8; i++ {
			out[i] = 0
		}
		binary.LittleEndian.PutUint32(out[4:8], crc32IEEE(out))
	}
	return out
}

func buildDataFirst(channelID, block, ack int, message []byte) []byte {
	packet := make([]byte, 20, 20+len(message))
	packet[0] = l4Data
	packet[1] = l4FlagFirst | l4FlagRequest
	binary.LittleEndian.PutUint16(packet[2:4], uint16(channelID))
	binary.LittleEndian.PutUint32(packet[4:8], uint32(block))
	binary.LittleEndian.PutUint32(packet[8:12], uint32(ack))
	binary.LittleEndian.PutUint32(packet[12:16], uint32(len(message)))
	binary.LittleEndian.PutUint32(packet[16:20], crc32IEEE(message))
	return append(packet, message...)
}

func buildData(channelID, block, ack int, chunk []byte) []byte {
	packet := make([]byte, 12, 12+len(chunk))
	packet[0] = l4Data
	packet[1] = l4FlagRequest
	binary.LittleEndian.PutUint16(packet[2:4], uint16(channelID))
	binary.LittleEndian.PutUint32(packet[4:8], uint32(block))
	binary.LittleEndian.PutUint32(packet[8:12], uint32(ack))
	return append(packet, chunk...)
}

func parseDataFirst(payload []byte) (data []byte, length int, checksum uint32, block int, err error) {
	if len(payload) < 20 {
		return nil, 0, 0, 0, fmt.Errorf("short CODESYS L4DATAFIRST")
	}
	block = int(binary.LittleEndian.Uint32(payload[4:8]))
	length = int(binary.LittleEndian.Uint32(payload[12:16]))
	checksum = binary.LittleEndian.Uint32(payload[16:20])
	return payload[20:], length, checksum, block, nil
}

func parseData(payload []byte) (data []byte, block int, err error) {
	if len(payload) < 12 {
		return nil, 0, fmt.Errorf("short CODESYS L4DATA")
	}
	block = int(binary.LittleEndian.Uint32(payload[4:8]))
	return payload[12:], block, nil
}

func buildService(group, service, sessionID int, tlv []byte) []byte {
	header := make([]byte, 20)
	binary.LittleEndian.PutUint16(header[0:2], protoID)
	binary.LittleEndian.PutUint16(header[2:4], 16)
	binary.LittleEndian.PutUint16(header[4:6], uint16(group))
	binary.LittleEndian.PutUint16(header[6:8], uint16(service))
	binary.LittleEndian.PutUint32(header[8:12], uint32(sessionID))
	binary.LittleEndian.PutUint32(header[12:16], uint32(len(tlv)))
	return append(header, tlv...)
}

func parseService(message []byte) (group, service, sessionID int, tags []byte, err error) {
	if len(message) < 20 {
		return 0, 0, 0, nil, fmt.Errorf("short CODESYS service message")
	}
	if binary.LittleEndian.Uint16(message[0:2]) != protoID {
		return 0, 0, 0, nil, fmt.Errorf("invalid CODESYS service protocol")
	}
	group = int(binary.LittleEndian.Uint16(message[4:6]) &^ serviceReplyBit)
	service = int(binary.LittleEndian.Uint16(message[6:8]))
	sessionID = int(binary.LittleEndian.Uint32(message[8:12]))
	length := int(binary.LittleEndian.Uint32(message[12:16]))
	start := 20
	if start+length > len(message) {
		return 0, 0, 0, nil, fmt.Errorf("short CODESYS service payload")
	}
	return group, service, sessionID, message[start : start+length], nil
}
