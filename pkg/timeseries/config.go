// Package timeseries provides InfluxDB 3 storage for thing time-series data.
package timeseries

import "time"

// Config controls the InfluxDB 3 Core connection and write pipeline.
type Config struct {
	Enabled            bool     `json:"enabled"`
	URL                string   `json:"url"`
	Token              string   `json:"token"`
	Database           string   `json:"database"`
	DebugDatabase      string   `json:"debug-database"`
	LogDatabase        string   `json:"log-database"`
	DataRetention      string   `json:"timeseries_data_retention"`
	DebugDataRetention string   `json:"debug-data-retention"`
	LogRetention       string   `json:"timeseries_log_retention"`
	BatchSize          int      `json:"batch-size"`
	FlushIntervalMs    int      `json:"flush-interval-ms"`
	QueueSize          int      `json:"queue-size"`
	WriteTimeoutMs     int      `json:"write-timeout-ms"`
	QueryAggregates    []string `json:"query-aggregates"`
}

// DefaultConfig returns the built-in defaults.
func DefaultConfig() Config {
	return Config{
		URL:                "http://127.0.0.1:8181",
		Database:           "harns_timeseries",
		DebugDatabase:      "harns_timeseries_debug",
		LogDatabase:        "harns_timeseries_log",
		DataRetention:      "730d",
		DebugDataRetention: "7d",
		LogRetention:       "1d",
		BatchSize:          10000,
		FlushIntervalMs:    3000,
		QueueSize:          20000,
		WriteTimeoutMs:     10000,
		QueryAggregates:    []string{"min", "max", "avg", "count", "sum", "median", "stddev"},
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
