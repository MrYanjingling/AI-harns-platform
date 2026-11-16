package licensemgr

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const licenseTableSQL = `
CREATE TABLE IF NOT EXISTS license (
    id   TEXT PRIMARY KEY,
    data TEXT NOT NULL
);
`

// Store persists the current license in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a license store.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Initialize creates the license table.
func (s *Store) Initialize(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("license store database is nil")
	}
	_, err := s.db.ExecContext(ctx, licenseTableSQL)
	return err
}

// Get returns the persisted license data.
func (s *Store) Get(ctx context.Context) (string, error) {
	if s == nil || s.db == nil {
		return "", fmt.Errorf("license store database is nil")
	}
	var data string
	err := s.db.QueryRowContext(ctx,
		`SELECT data FROM license WHERE id = ?`, LicenseRecordID).Scan(&data)
	if err != nil {
		return "", err
	}
	return data, nil
}

// GatewayInitTime returns the persisted gateway initialization time.
func (s *Store) GatewayInitTime(ctx context.Context) (time.Time, error) {
	if s == nil || s.db == nil {
		return time.Time{}, fmt.Errorf("license store database is nil")
	}
	var raw string
	if err := s.db.QueryRowContext(ctx,
		`SELECT init_time FROM gateway ORDER BY id LIMIT 1`).Scan(&raw); err != nil {
		return time.Time{}, err
	}
	initTime, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid gateway init_time %q: %w", raw, err)
	}
	return initTime, nil
}

// Save replaces the persisted license data.
func (s *Store) Save(ctx context.Context, data string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("license store database is nil")
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO license (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		LicenseRecordID, data)
	return err
}
