package aimgr

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
)

const (
	aiPayloadVersion = byte(1)
	aiPayloadAAD     = "harns-gateway-ai-v1"
)

type codec struct {
	aead cipher.AEAD
}

func newCodec(secret string) (*codec, error) {
	if secret == "" {
		return nil, fmt.Errorf("ai encryption secret is empty")
	}
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("create ai cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create ai GCM: %w", err)
	}
	return &codec{aead: aead}, nil
}

func (c *codec) encryptString(value string) ([]byte, error) {
	if value == "" {
		return nil, nil
	}
	nonceSize := c.aead.NonceSize()
	data := make([]byte, 1+nonceSize, 1+nonceSize+len(value)+c.aead.Overhead())
	data[0] = aiPayloadVersion
	if _, err := rand.Read(data[1 : 1+nonceSize]); err != nil {
		return nil, fmt.Errorf("generate ai nonce: %w", err)
	}
	return c.aead.Seal(data, data[1:1+nonceSize], []byte(value), []byte(aiPayloadAAD)), nil
}

func (c *codec) decryptString(data []byte) (string, error) {
	if len(data) == 0 {
		return "", nil
	}
	nonceSize := c.aead.NonceSize()
	if len(data) < 1+nonceSize+c.aead.Overhead() {
		return "", fmt.Errorf("ai payload is too short")
	}
	if data[0] != aiPayloadVersion {
		return "", fmt.Errorf("unsupported ai payload version %d", data[0])
	}
	plain, err := c.aead.Open(nil, data[1:1+nonceSize], data[1+nonceSize:], []byte(aiPayloadAAD))
	if err != nil {
		return "", fmt.Errorf("decrypt ai payload: %w", err)
	}
	return string(plain), nil
}
