package gatewaymgr

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
)

const (
	gatewayMetaKey     = "abcdefghijklmnopqrstuvwxyzycbyjl"
	gatewayMetaAAD     = "harns-gateway-meta-v1"
	gatewayMetaVersion = byte(1)
)

type metaCodec struct {
	aead cipher.AEAD
}

func newMetaCodec(key string) (*metaCodec, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("gateway meta key must be 32 bytes")
	}
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("create gateway meta cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gateway meta GCM: %w", err)
	}
	return &metaCodec{aead: aead}, nil
}

func (c *metaCodec) encryptJSON(value any) ([]byte, error) {
	plain, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal gateway meta: %w", err)
	}
	nonceSize := c.aead.NonceSize()
	data := make([]byte, 1+nonceSize, 1+nonceSize+len(plain)+c.aead.Overhead())
	data[0] = gatewayMetaVersion
	if _, err := rand.Read(data[1 : 1+nonceSize]); err != nil {
		return nil, fmt.Errorf("generate gateway meta nonce: %w", err)
	}
	return c.aead.Seal(data, data[1:1+nonceSize], plain, []byte(gatewayMetaAAD)), nil
}

func (c *metaCodec) decryptJSON(data []byte, value any) error {
	nonceSize := c.aead.NonceSize()
	if len(data) < 1+nonceSize+c.aead.Overhead() {
		return fmt.Errorf("gateway meta payload is too short")
	}
	if data[0] != gatewayMetaVersion {
		return fmt.Errorf("unsupported gateway meta payload version %d", data[0])
	}
	plain, err := c.aead.Open(nil, data[1:1+nonceSize], data[1+nonceSize:], []byte(gatewayMetaAAD))
	if err != nil {
		return fmt.Errorf("decrypt gateway meta: %w", err)
	}
	if err := json.Unmarshal(plain, value); err != nil {
		return fmt.Errorf("unmarshal gateway meta: %w", err)
	}
	return nil
}
