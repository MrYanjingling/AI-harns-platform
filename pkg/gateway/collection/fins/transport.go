package fins

import (
	"bufio"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"harns-platform/pkg/gateway/collection/connpool"
)

const tcpMagic = "FINS"

var (
	_ connpool.Transport = (*tcpTransport)(nil)
	_ connpool.Transport = (*udpTransport)(nil)
	_ connpool.Transport = (*hostlinkTransport)(nil)
)

type tcpTransport struct {
	conn    net.Conn
	timeout time.Duration
	mu      sync.Mutex
	srcNode byte
	dstNode byte
	sid     byte
}

func newTCPTransport(ctx context.Context, address string, timeout time.Duration) (*tcpTransport, error) {
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	client := &tcpTransport{conn: conn, timeout: timeout}
	clientNode, serverNode, err := client.handshake()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	client.srcNode = clientNode
	client.dstNode = serverNode
	return client, nil
}

func (t *tcpTransport) handshake() (byte, byte, error) {
	if err := t.conn.SetDeadline(time.Now().Add(t.timeout)); err != nil {
		return 0, 0, err
	}
	request := wrapTCP(0, []byte{0, 0, 0, 0})
	if _, err := t.conn.Write(request); err != nil {
		return 0, 0, err
	}
	command, errorCode, payload, err := readTCPPacket(t.conn)
	if err != nil {
		return 0, 0, err
	}
	if command != 1 {
		return 0, 0, fmt.Errorf("fins/tcp handshake command is %d", command)
	}
	if errorCode != 0 {
		return 0, 0, fmt.Errorf("fins/tcp handshake error %d", errorCode)
	}
	if len(payload) < 8 {
		return 0, 0, fmt.Errorf("fins/tcp handshake response is too short")
	}
	return payload[3], payload[7], nil
}

func (t *tcpTransport) Exchange(ctx context.Context, request []byte) ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn == nil {
		return nil, fmt.Errorf("fins/tcp connection is closed")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	t.sid++
	frame := append([]byte(nil), request...)
	if len(frame) < 10 {
		return nil, fmt.Errorf("fins request is too short")
	}
	frame[4] = t.dstNode
	frame[7] = t.srcNode
	frame[9] = t.sid
	if err := t.conn.SetDeadline(time.Now().Add(t.timeout)); err != nil {
		return nil, err
	}
	if _, err := t.conn.Write(wrapTCP(2, frame)); err != nil {
		return nil, err
	}
	command, errorCode, payload, err := readTCPPacket(t.conn)
	if err != nil {
		return nil, err
	}
	if command != 2 {
		return nil, fmt.Errorf("fins/tcp data command is %d", command)
	}
	if errorCode != 0 {
		return nil, fmt.Errorf("fins/tcp data error %d", errorCode)
	}
	return payload, nil
}

func (t *tcpTransport) Available() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.conn != nil
}

func (t *tcpTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn == nil {
		return nil
	}
	conn := t.conn
	t.conn = nil
	return conn.Close()
}

func wrapTCP(command uint32, payload []byte) []byte {
	frame := make([]byte, 16+len(payload))
	copy(frame[:4], tcpMagic)
	length := uint32(8 + len(payload))
	frame[4] = byte(length >> 24)
	frame[5] = byte(length >> 16)
	frame[6] = byte(length >> 8)
	frame[7] = byte(length)
	frame[8] = byte(command >> 24)
	frame[9] = byte(command >> 16)
	frame[10] = byte(command >> 8)
	frame[11] = byte(command)
	copy(frame[16:], payload)
	return frame
}

func readTCPPacket(conn net.Conn) (uint32, uint32, []byte, error) {
	header := make([]byte, 16)
	if _, err := io.ReadFull(conn, header); err != nil {
		return 0, 0, nil, err
	}
	if string(header[:4]) != tcpMagic {
		return 0, 0, nil, fmt.Errorf("fins/tcp magic mismatch")
	}
	length := int(uint32(header[4])<<24 | uint32(header[5])<<16 | uint32(header[6])<<8 | uint32(header[7]))
	if length < 8 || length > 65535 {
		return 0, 0, nil, fmt.Errorf("invalid fins/tcp length %d", length)
	}
	command := uint32(header[8])<<24 | uint32(header[9])<<16 | uint32(header[10])<<8 | uint32(header[11])
	errorCode := uint32(header[12])<<24 | uint32(header[13])<<16 | uint32(header[14])<<8 | uint32(header[15])
	if command == 3 {
		return 0, 0, nil, fmt.Errorf("fins/tcp frame error %d", errorCode)
	}
	payload := make([]byte, length-8)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return 0, 0, nil, err
	}
	return command, errorCode, payload, nil
}

type udpTransport struct {
	conn    net.Conn
	timeout time.Duration
	mu      sync.Mutex
	dstNode byte
	sid     byte
}

func newUDPTransport(ctx context.Context, host string, port int, timeout time.Duration) (*udpTransport, error) {
	dialer := net.Dialer{Timeout: timeout}
	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := dialer.DialContext(ctx, "udp", address)
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(host)
	var node byte
	if ip != nil {
		if v4 := ip.To4(); v4 != nil {
			node = v4[3]
		}
	}
	return &udpTransport{conn: conn, timeout: timeout, dstNode: node}, nil
}

func (t *udpTransport) Exchange(ctx context.Context, request []byte) ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn == nil {
		return nil, fmt.Errorf("fins/udp connection is closed")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	t.sid++
	frame := append([]byte(nil), request...)
	if len(frame) < 10 {
		return nil, fmt.Errorf("fins request is too short")
	}
	frame[4] = t.dstNode
	frame[7] = 0
	frame[9] = t.sid
	if err := t.conn.SetDeadline(time.Now().Add(t.timeout)); err != nil {
		return nil, err
	}
	if _, err := t.conn.Write(frame); err != nil {
		return nil, err
	}
	buffer := make([]byte, 65535)
	length, err := t.conn.Read(buffer)
	if err != nil {
		return nil, err
	}
	return buffer[:length], nil
}

func (t *udpTransport) Available() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.conn != nil
}

func (t *udpTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn == nil {
		return nil
	}
	conn := t.conn
	t.conn = nil
	return conn.Close()
}

// hostlinkTransport wraps FINS command/response frames inside Omron W342
// HostLink C-Mode FA framing: @<unit:02d>FA0<ICF><DA2><SA2><SID><FINS><FCS>*\r
type hostlinkTransport struct {
	conn    net.Conn
	reader  *bufio.Reader
	unit    int
	timeout time.Duration
	mu      sync.Mutex
	sid     byte
}

func newHostlinkTransport(ctx context.Context, address string, unit int, timeout time.Duration) (*hostlinkTransport, error) {
	if unit < 0 || unit > 31 {
		unit = 0
	}
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	return &hostlinkTransport{
		conn:    conn,
		reader:  bufio.NewReader(conn),
		unit:    unit,
		timeout: timeout,
	}, nil
}

func (t *hostlinkTransport) Exchange(ctx context.Context, request []byte) ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn == nil {
		return nil, fmt.Errorf("fins/hostlink connection is closed")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(request) < 12 {
		return nil, fmt.Errorf("fins request is too short")
	}
	t.sid++
	frame := buildHostlinkFINSRequest(t.unit, request[0], request[5], request[8], t.sid, request[10:])
	if err := t.conn.SetDeadline(time.Now().Add(t.timeout)); err != nil {
		return nil, err
	}
	if _, err := t.conn.Write(frame); err != nil {
		return nil, err
	}
	line, err := t.reader.ReadString('\r')
	if err != nil {
		return nil, err
	}
	return parseHostlinkFINSResponse(line)
}

func (t *hostlinkTransport) Available() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.conn != nil
}

func (t *hostlinkTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn == nil {
		return nil
	}
	conn := t.conn
	t.conn = nil
	return conn.Close()
}

func hostlinkFCS(data string) byte {
	var fcs byte
	for i := 0; i < len(data); i++ {
		fcs ^= data[i]
	}
	return fcs
}

func buildHostlinkFINSRequest(unit int, icf, da2, sa2, sid byte, finsBody []byte) []byte {
	body := fmt.Sprintf("@%02dFA0%02X%02X%02X%02X%s", unit, icf, da2, sa2, sid, strings.ToUpper(hex.EncodeToString(finsBody)))
	return []byte(fmt.Sprintf("%s%02X*\r", body, hostlinkFCS(body)))
}

func parseHostlinkFINSResponse(raw string) ([]byte, error) {
	text := strings.TrimRight(raw, "\r\n")
	if len(text) < 22 || text[0] != '@' || !strings.HasSuffix(text, "*") {
		return nil, fmt.Errorf("invalid hostlink FINS response %q", text)
	}
	withoutTerm := text[:len(text)-1] // strip '*'
	payload := withoutTerm[:len(withoutTerm)-2]
	fcsHex := withoutTerm[len(withoutTerm)-2:]
	expected := fmt.Sprintf("%02X", hostlinkFCS(payload))
	if !strings.EqualFold(fcsHex, expected) {
		return nil, fmt.Errorf("hostlink FINS FCS mismatch: got %s, want %s", fcsHex, expected)
	}
	if len(payload) < 15 || payload[3:5] != "FA" {
		return nil, fmt.Errorf("invalid hostlink FINS header %q", payload)
	}
	endCode := payload[5:7]
	if endCode != "00" {
		return nil, fmt.Errorf("hostlink FINS end code %s", endCode)
	}
	hdrHex := payload[7:15]
	hdrBytes, err := hex.DecodeString(hdrHex)
	if err != nil || len(hdrBytes) != 4 {
		return nil, fmt.Errorf("invalid hostlink FINS header hex %q", hdrHex)
	}
	finsBody, err := hex.DecodeString(payload[15:])
	if err != nil || len(finsBody) < 4 {
		return nil, fmt.Errorf("invalid hostlink FINS body: %w", err)
	}
	// Reconstruct standard 10-byte FINS response header + MRC/SRC/MRES/SRES/data.
	out := make([]byte, 10+len(finsBody))
	out[0] = hdrBytes[0] // ICF
	out[2] = 0x02        // GCT
	out[5] = hdrBytes[1] // DA2
	out[8] = hdrBytes[2] // SA2
	out[9] = hdrBytes[3] // SID
	copy(out[10:], finsBody)
	return out, nil
}
