package knx

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"time"
)

const (
	serviceSearchRequest       = 0x0201
	serviceSearchResponse      = 0x0202
	serviceConnectRequest      = 0x0205
	serviceConnectResponse     = 0x0206
	serviceConnectionState     = 0x0207
	serviceConnectionStateResp = 0x0208
	serviceTunnelingRequest    = 0x0420
	serviceTunnelingAck        = 0x0421
)

type knxClient struct {
	conn         *net.UDPConn
	gateway      *net.UDPAddr
	dataEndpoint *net.UDPAddr
	timeout      time.Duration
	channel      byte
	sendSeq      byte
	recvSeq      byte
	closed       bool
}

func openKNXClient(discoveryIP string, discoveryPort int, timeout time.Duration) (*knxClient, error) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, err
	}
	client := &knxClient{conn: conn, timeout: timeout}
	if err := client.discover(discoveryIP, discoveryPort); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := client.connect(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}

func (c *knxClient) Available() bool { return c != nil && c.conn != nil && !c.closed }
func (c *knxClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	c.closed = true
	return c.conn.Close()
}

func (c *knxClient) discover(ip string, port int) error {
	destination := &net.UDPAddr{IP: net.ParseIP(ip), Port: port}
	if destination.IP == nil {
		return fmt.Errorf("invalid KNX discovery IP %q", ip)
	}
	local := c.localAddressFor(destination)
	request := makeHeader(serviceSearchRequest, 14)
	request = append(request, encodeHPAI(local)...)
	if err := c.writePacket(request, destination); err != nil {
		return err
	}
	deadline := time.Now().Add(c.timeout)
	for {
		packet, source, err := c.readPacket(deadline)
		if err != nil {
			// Strict KNXnet/IP servers (e.g. Calimero) only accept SEARCH_REQUEST (0x0201)
			// on the multicast discovery endpoint (224.0.23.12:3671) and ignore it on the
			// unicast control port. When the user specifies a unicast gateway IP directly,
			// fall back to connecting to that control endpoint directly.
			if !destination.IP.IsMulticast() {
				c.gateway = destination
				return nil
			}
			return err
		}
		service, total, err := parseHeader(packet)
		if err != nil || total > len(packet) || service != serviceSearchResponse {
			continue
		}
		endpoint, err := parseHPAI(packet[6:])
		if err != nil {
			continue
		}
		if endpoint.IP == nil || endpoint.IP.IsUnspecified() {
			endpoint.IP = source.IP
		}
		c.gateway = endpoint
		return nil
	}
}

func (c *knxClient) connect() error {
	if c.gateway == nil {
		return fmt.Errorf("KNX gateway is not discovered")
	}
	local := c.localAddressFor(c.gateway)
	request := makeHeader(serviceConnectRequest, 26)
	request = append(request, encodeHPAI(local)...)
	request = append(request, encodeHPAI(local)...)
	request = append(request, 0x04, 0x04, 0x02, 0x00)
	if err := c.writePacket(request, c.gateway); err != nil {
		return err
	}
	deadline := time.Now().Add(c.timeout)
	for {
		packet, _, err := c.readPacket(deadline)
		if err != nil {
			return err
		}
		service, total, err := parseHeader(packet)
		if err != nil || total > len(packet) || service != serviceConnectResponse {
			continue
		}
		if len(packet) < 8 {
			return fmt.Errorf("short KNX connect response")
		}
		if packet[7] != 0 {
			return fmt.Errorf("KNX connect failed with status 0x%02X", packet[7])
		}
		c.channel = packet[6]
		if len(packet) >= 16 {
			if endpoint, err := parseHPAI(packet[8:]); err == nil && endpoint.IP != nil && !endpoint.IP.IsUnspecified() {
				c.dataEndpoint = endpoint
			}
		}
		if c.dataEndpoint == nil {
			c.dataEndpoint = c.gateway
		}
		return nil
	}
}

func (c *knxClient) Read(group uint16) ([]byte, error) {
	destination := c.dataEndpoint
	if destination == nil {
		destination = c.gateway
	}
	if err := c.writePacket(c.buildTunnelRead(group), destination); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(c.timeout)
	for {
		packet, _, err := c.readPacket(deadline)
		if err != nil {
			return nil, err
		}
		service, total, err := parseHeader(packet)
		if err != nil || total > len(packet) {
			continue
		}
		switch service {
		case serviceTunnelingAck:
			if len(packet) >= 10 && packet[7] == c.channel && packet[8] == c.sendSeq {
				c.sendSeq++
			}
		case serviceTunnelingRequest:
			if len(packet) < 10 {
				continue
			}
			if packet[7] != c.channel {
				continue
			}
			_ = c.writePacket(buildTunnelAck(c.channel, packet[8]), destination)
			if packet[8] == c.recvSeq {
				c.recvSeq++
			}
			destination, data, messageCode, apci, err := parseCEMI(packet)
			if err != nil {
				continue
			}
			if destination != group || apci != 1 {
				continue
			}
			if messageCode != 0x29 && messageCode != 0x2e {
				continue
			}
			return data, nil
		case serviceConnectionStateResp:
			continue
		}
	}
}

func (c *knxClient) buildTunnelRead(group uint16) []byte {
	cemi := []byte{0x11, 0x00, 0xbc, 0xe0, 0x00, 0x00, byte(group >> 8), byte(group), 0x01, 0x00, 0x00}
	request := makeHeader(serviceTunnelingRequest, 21)
	request = append(request, 0x04, c.channel, c.sendSeq, 0x00)
	return append(request, cemi...)
}

func buildTunnelAck(channel, sequence byte) []byte {
	ack := makeHeader(serviceTunnelingAck, 10)
	return append(ack, 0x04, channel, sequence, 0x00)
}

func (c *knxClient) localAddressFor(destination *net.UDPAddr) *net.UDPAddr {
	port := 0
	var localIP net.IP
	if c.conn != nil {
		if address, ok := c.conn.LocalAddr().(*net.UDPAddr); ok {
			port = address.Port
			if ip := address.IP.To4(); ip != nil && !ip.IsUnspecified() {
				localIP = ip
			}
		}
	}
	if localIP == nil && destination != nil && destination.IP != nil {
		if probe, err := net.DialUDP("udp4", nil, destination); err == nil {
			if addr, ok := probe.LocalAddr().(*net.UDPAddr); ok {
				if ip := addr.IP.To4(); ip != nil && !ip.IsUnspecified() {
					localIP = ip
				}
			}
			_ = probe.Close()
		}
	}
	if localIP == nil {
		localIP = firstNonLoopbackIPv4()
	}
	if localIP == nil {
		localIP = net.IPv4(127, 0, 0, 1)
	}
	return &net.UDPAddr{IP: localIP, Port: port}
}

func firstNonLoopbackIPv4() net.IP {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			if ip := ipNet.IP.To4(); ip != nil && !ip.IsLoopback() && !ip.IsUnspecified() {
				return ip
			}
		}
	}
	return nil
}

func (c *knxClient) writePacket(packet []byte, destination *net.UDPAddr) error {
	if destination == nil {
		return fmt.Errorf("KNX destination is empty")
	}
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return err
	}
	_, err := c.conn.WriteToUDP(packet, destination)
	return err
}

func (c *knxClient) readPacket(deadline time.Time) ([]byte, *net.UDPAddr, error) {
	if err := c.conn.SetReadDeadline(deadline); err != nil {
		return nil, nil, err
	}
	buffer := make([]byte, 4096)
	n, source, err := c.conn.ReadFromUDP(buffer)
	if err != nil {
		return nil, nil, err
	}
	return buffer[:n], source, nil
}

func makeHeader(service uint16, total int) []byte {
	header := make([]byte, 6)
	header[0] = 0x06
	header[1] = 0x10
	binary.BigEndian.PutUint16(header[2:4], service)
	binary.BigEndian.PutUint16(header[4:6], uint16(total))
	return header
}

func parseHeader(packet []byte) (uint16, int, error) {
	if len(packet) < 6 || packet[0] != 0x06 || packet[1] != 0x10 {
		return 0, 0, fmt.Errorf("invalid KNXnet/IP header")
	}
	return binary.BigEndian.Uint16(packet[2:4]), int(binary.BigEndian.Uint16(packet[4:6])), nil
}

func encodeHPAI(address *net.UDPAddr) []byte {
	ip := address.IP.To4()
	if ip == nil {
		ip = net.IPv4zero.To4()
	}
	hpai := make([]byte, 8)
	hpai[0] = 0x08
	hpai[1] = 0x01
	copy(hpai[2:6], ip)
	binary.BigEndian.PutUint16(hpai[6:8], uint16(address.Port))
	return hpai
}

func parseHPAI(data []byte) (*net.UDPAddr, error) {
	if len(data) < 8 || data[0] != 0x08 || (data[1] != 0x01 && data[1] != 0x02) {
		return nil, fmt.Errorf("invalid KNX HPAI")
	}
	return &net.UDPAddr{IP: net.IPv4(data[2], data[3], data[4], data[5]), Port: int(binary.BigEndian.Uint16(data[6:8]))}, nil
}

func parseCEMI(packet []byte) (uint16, []byte, byte, int, error) {
	if len(packet) < 12 {
		return 0, nil, 0, 0, fmt.Errorf("short KNX tunneling frame")
	}
	offset := 10
	messageCode := packet[offset]
	additional := int(packet[offset+1])
	base := offset + 2 + additional
	if base+8 > len(packet) {
		return 0, nil, 0, 0, fmt.Errorf("short KNX cEMI")
	}
	npduLength := int(packet[base+6])
	if npduLength < 1 || base+8+npduLength > len(packet) {
		return 0, nil, 0, 0, fmt.Errorf("short KNX NPDU")
	}
	destination := binary.BigEndian.Uint16(packet[base+4 : base+6])
	data := packet[base+8 : base+8+npduLength]
	apci := int(data[0]>>6) | int(packet[base+7]&0x03)<<2
	return destination, data, messageCode, apci, nil
}

func decodeValue(data []byte, address Address, dataType string) (any, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty KNX data")
	}
	switch dataType {
	case "bit", "BIT":
		if data[0]&0x01 != 0 {
			return 1, nil
		}
		return 0, nil
	case "bool", "BOOL":
		return data[0]&0x01 != 0, nil
	case "uint8", "UINT8":
		if address.NBit > 0 {
			return data[0] & byte((1<<address.NBit)-1), nil
		}
		if len(data) < 2 {
			return nil, fmt.Errorf("short KNX uint8 data")
		}
		return data[1], nil
	case "int8", "INT8":
		if len(data) < 2 {
			return nil, fmt.Errorf("short KNX int8 data")
		}
		return int8(data[1]), nil
	case "uint16", "UINT16":
		if len(data) < 3 {
			return nil, fmt.Errorf("short KNX uint16 data")
		}
		return binary.BigEndian.Uint16(data[1:3]), nil
	case "int16", "INT16":
		if len(data) < 3 {
			return nil, fmt.Errorf("short KNX int16 data")
		}
		return int16(binary.BigEndian.Uint16(data[1:3])), nil
	case "uint32", "UINT32":
		if len(data) < 5 {
			return nil, fmt.Errorf("short KNX uint32 data")
		}
		return binary.BigEndian.Uint32(data[1:5]), nil
	case "int32", "INT32":
		if len(data) < 5 {
			return nil, fmt.Errorf("short KNX int32 data")
		}
		return int32(binary.BigEndian.Uint32(data[1:5])), nil
	case "float", "FLOAT", "float32", "FLOAT32":
		if len(data) < 5 {
			return nil, fmt.Errorf("short KNX float data")
		}
		return math.Float32frombits(binary.BigEndian.Uint32(data[1:5])), nil
	default:
		return nil, fmt.Errorf("unsupported KNX data type %q", dataType)
	}
}
