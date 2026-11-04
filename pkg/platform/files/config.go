// Package files provides generic file upload services backed by MinIO.
package files

// Config controls file upload limits.
type Config struct {
	MaxImageBytes int64 `json:"max-image-bytes"`
}

// DefaultConfig returns the built-in defaults.
func DefaultConfig() Config {
	return Config{MaxImageBytes: 10 << 20}
}
