package collection

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
)

// ErrConnectionLost reports a transport-level failure that should trigger reconnect.
var ErrConnectionLost = errors.New("collection connection lost")

// ErrReadTimeout reports that a read timed out before any data arrived.
// Serial ports return (0, nil) when SetReadTimeout expires.
var ErrReadTimeout = errors.New("serial read timeout")

// ReadFullWithTimeout reads exactly len(buf) bytes and treats a zero-byte read
// without error as a timeout.
func ReadFullWithTimeout(r io.Reader, buf []byte) error {
	for read := 0; read < len(buf); {
		n, err := r.Read(buf[read:])
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrReadTimeout
		}
		read += n
	}
	return nil
}

// ConnectionLost wraps err as a retryable connection failure.
func ConnectionLost(err error) error {
	if err == nil || errors.Is(err, ErrConnectionLost) {
		return err
	}
	return fmt.Errorf("%w: %w", ErrConnectionLost, err)
}

// IsConnectionLost reports whether err is a transport-level failure.
func IsConnectionLost(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrConnectionLost) {
		return true
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, net.ErrClosed) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	if IsTimeout(err) {
		return true
	}
	text := strings.ToLower(err.Error())
	for _, marker := range []string{
		"connection reset",
		"connection refused",
		"broken pipe",
		"closed",
		"not connected",
		"port not found",
		"serial port not found",
		"no route to host",
		"network is unreachable",
		"unexpected eof",
		"eof",
		"port is closed",
		"device not configured",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

// IsTimeout reports whether err is a network or serial timeout.
func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "timeout") || strings.Contains(text, "deadline exceeded")
}
