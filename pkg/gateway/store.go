package gateway

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"harns-platform/pkg/gateway/runtime"
)

const gatewayTableSQL = `
CREATE TABLE IF NOT EXISTS gateway (
    id       TEXT PRIMARY KEY,
    name     TEXT NOT NULL,
    version  TEXT NOT NULL,
    secret    TEXT NOT NULL DEFAULT '',
    init_time TEXT NOT NULL,
    mod_time  TEXT NOT NULL
);
`

// Store persists gateway metadata in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a gateway metadata store.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Initialize creates the gateway metadata table.
func (s *Store) Initialize(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("gateway store database is nil")
	}
	_, err := s.db.ExecContext(ctx, gatewayTableSQL)
	return err
}

// GetMeta returns the persisted gateway metadata.
func (s *Store) GetMeta(ctx context.Context) (*GatewayMeta, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("gateway store database is nil")
	}
	var (
		id       string
		name     string
		version  string
		secret   string
		initTime string
		modTime  string
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, version, secret, init_time, mod_time FROM gateway ORDER BY id LIMIT 1`).
		Scan(&id, &name, &version, &secret, &initTime, &modTime)
	if err != nil {
		return nil, err
	}
	initTS, err := time.Parse(time.RFC3339Nano, initTime)
	if err != nil {
		return nil, fmt.Errorf("invalid gateway init_time %q: %w", initTime, err)
	}
	modTS, err := time.Parse(time.RFC3339Nano, modTime)
	if err != nil {
		return nil, fmt.Errorf("invalid gateway mod_time %q: %w", modTime, err)
	}
	return &GatewayMeta{
		Secret:   secret,
		InitTime: initTS,
		ObjectMeta: runtime.ObjectMeta{
			Name:    name,
			ID:      id,
			Version: version,
			ModTime: modTS,
		},
	}, nil
}

// CreateMeta inserts gateway metadata.
func (s *Store) CreateMeta(ctx context.Context, meta *GatewayMeta) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("gateway store database is nil")
	}
	if meta == nil {
		return fmt.Errorf("gateway meta is nil")
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO gateway (id, name, version, secret, init_time, mod_time) VALUES (?, ?, ?, ?, ?, ?)`,
		meta.ID, meta.Name, meta.Version, meta.Secret,
		meta.InitTime.UTC().Format(time.RFC3339Nano), meta.ModTime.UTC().Format(time.RFC3339Nano))
	return err
}

// UpdateMeta updates the current startup time and code version.
func (s *Store) UpdateMeta(ctx context.Context, meta *GatewayMeta) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("gateway store database is nil")
	}
	if meta == nil {
		return fmt.Errorf("gateway meta is nil")
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE gateway SET version = ?, secret = ?, mod_time = ? WHERE id = ?`,
		meta.Version, meta.Secret, meta.ModTime.UTC().Format(time.RFC3339Nano), meta.ID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
