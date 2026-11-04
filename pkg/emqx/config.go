// Package emqx provides a client for the EMQX HTTP API.
package emqx

import "time"

// Config controls the EMQX API connection.
type Config struct {
	APIURL    string `json:"api-url"`
	AK        string `json:"ak"`
	SK        string `json:"sk"`
	Broker    string `json:"broker"`
	TimeoutMs int    `json:"timeout-ms"`
}

// DefaultConfig returns the built-in defaults.
func DefaultConfig() Config {
	return Config{
		APIURL:    "http://127.0.0.1:18083",
		Broker:    "127.0.0.1:1883",
		TimeoutMs: 10000,
	}
}

// Timeout returns the configured HTTP timeout.
func (c Config) Timeout() time.Duration {
	if c.TimeoutMs <= 0 {
		return 10 * time.Second
	}
	return time.Duration(c.TimeoutMs) * time.Millisecond
}
