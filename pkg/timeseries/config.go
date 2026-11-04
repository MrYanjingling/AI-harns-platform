// Package timeseries provides InfluxDB 3 storage for thing time-series data.
package timeseries

import "time"

// Config controls the InfluxDB 3 Core connection and write pipeline.
type Config struct {
	Enabled         bool   `json:"enabled"`
	URL             string `json:"url"`
	Token           string `json:"token"`
	Database        string `json:"database"`
	LogDatabase     string `json:"log-database"`
	DataRetention   string `json:"timeseries_data_retention"`
	LogRetention    string `json:"timeseries_log_retention"`
	BatchSize       int    `json:"batch-size"`
	FlushIntervalMs int    `json:"flush-interval-ms"`
	QueueSize       int    `json:"queue-size"`
	WriteTimeoutMs  int    `json:"write-timeout-ms"`
}

// DefaultConfig returns the built-in defaults.
func DefaultConfig() Config {
	return Config{
		URL:             "http://127.0.0.1:8181",
		Database:        "harns_timeseries",
		LogDatabase:     "harns_timeseries_log",
		DataRetention:   "730d",
		LogRetention:    "1d",
		BatchSize:       10000,
		FlushIntervalMs: 3000,
		QueueSize:       20000,
		WriteTimeoutMs:  10000,
	}
}

// FlushInterval returns the configured flush interval.
func (c Config) FlushInterval() time.Duration {
	return time.Duration(c.FlushIntervalMs) * time.Millisecond
}

// WriteTimeout returns the configured write timeout.
func (c Config) WriteTimeout() time.Duration {
	return time.Duration(c.WriteTimeoutMs) * time.Millisecond
}
