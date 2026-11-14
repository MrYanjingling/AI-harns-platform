package licensemgr

import (
	"context"
	"database/sql"
	"fmt"
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
