package gatewaymgr

import (
	"context"
	"database/sql"
	"fmt"
)

const gatewayMetaTableSQL = `
CREATE TABLE IF NOT EXISTS gateway_meta (
    singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
    payload   BLOB NOT NULL
);
`

// Store persists encrypted gateway metadata in SQLite.
type Store struct {
	db    *sql.DB
	codec *metaCodec
}

// NewStore creates a gateway metadata store.
func NewStore(db *sql.DB) *Store {
	codec, err := newMetaCodec(gatewayMetaKey)
	if err != nil {
		panic(err)
	}
	return &Store{db: db, codec: codec}
}

// Initialize creates the encrypted gateway metadata table.
func (s *Store) Initialize(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("gateway store database is nil")
	}
	legacy, err := tableExists(ctx, s.db, "gateway")
	if err != nil {
		return err
	}
	if legacy {
		if _, err := s.db.ExecContext(ctx, `PRAGMA secure_delete = ON`); err != nil {
			return fmt.Errorf("enable secure delete: %w", err)
		}
		if _, err := s.db.ExecContext(ctx, `DROP TABLE gateway`); err != nil {
			return fmt.Errorf("drop legacy gateway table: %w", err)
		}
		if _, err := s.db.ExecContext(ctx, `VACUUM`); err != nil {
			return fmt.Errorf("vacuum legacy gateway table: %w", err)
		}
	}
	_, err = s.db.ExecContext(ctx, gatewayMetaTableSQL)
	return err
}

func tableExists(ctx context.Context, db *sql.DB, name string) (bool, error) {
	var count int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?`, name).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetMeta returns the persisted gateway metadata.
func (s *Store) GetMeta(ctx context.Context) (*GatewayMeta, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("gateway store database is nil")
	}
	var payload []byte
	err := s.db.QueryRowContext(ctx,
		`SELECT payload FROM gateway_meta WHERE singleton = 1`).Scan(&payload)
	if err != nil {
		return nil, err
	}
	meta := &GatewayMeta{}
	if err := s.codec.decryptJSON(payload, meta); err != nil {
		return nil, err
	}
	return meta, nil
}

// UpdateName updates the gateway display name.
func (s *Store) UpdateName(ctx context.Context, id, name, eTag string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("gateway store database is nil")
	}
	meta, err := s.GetMeta(ctx)
	if err != nil {
		return err
	}
	if meta.ID != id {
		return sql.ErrNoRows
	}
	meta.Name = name
	meta.Version = eTag
	return s.UpdateMeta(ctx, meta)
}

// CreateMeta inserts gateway metadata.
func (s *Store) CreateMeta(ctx context.Context, meta *GatewayMeta) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("gateway store database is nil")
	}
	if meta == nil {
		return fmt.Errorf("gateway meta is nil")
	}
	payload, err := s.codec.encryptJSON(meta)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO gateway_meta (singleton, payload) VALUES (1, ?)`, payload)
	return err
}

// UpdateMeta updates the current gateway metadata.
func (s *Store) UpdateMeta(ctx context.Context, meta *GatewayMeta) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("gateway store database is nil")
	}
	if meta == nil {
		return fmt.Errorf("gateway meta is nil")
	}
	payload, err := s.codec.encryptJSON(meta)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE gateway_meta SET payload = ? WHERE singleton = 1`, payload)
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
