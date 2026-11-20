package aimgr

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS ai_settings (
    singleton          INTEGER PRIMARY KEY CHECK (singleton = 1),
    assistant_enabled  INTEGER NOT NULL DEFAULT 0,
    mcp_enabled        INTEGER NOT NULL DEFAULT 0,
    name               TEXT NOT NULL DEFAULT '',
    remark             TEXT NOT NULL DEFAULT '',
    model              TEXT NOT NULL DEFAULT '',
    endpoint_url       TEXT NOT NULL DEFAULT '',
    api_key_cipher     BLOB,
    mcp_token          TEXT NOT NULL DEFAULT '',
    updated_time       TEXT NOT NULL,
    CHECK (assistant_enabled IN (0, 1)),
    CHECK (mcp_enabled IN (0, 1))
);
`

// Store persists AI settings in SQLite.
type Store struct {
	db    *sql.DB
	codec *codec
}

// NewStore creates an AI configuration store.
func NewStore(db *sql.DB, secret string) (*Store, error) {
	c, err := newCodec(secret)
	if err != nil {
		return nil, err
	}
	return &Store{db: db, codec: c}, nil
}

// Initialize creates AI tables and default settings.
func (s *Store) Initialize(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("ai store database is nil")
	}
	if _, err := s.db.ExecContext(ctx, schemaSQL); err != nil {
		return err
	}
	now := formatTime(time.Now().UTC())
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO ai_settings
		 (singleton, assistant_enabled, mcp_enabled, name, remark, model, endpoint_url, api_key_cipher, mcp_token, updated_time)
		 VALUES (1, 0, 0, '', '', '', '', NULL, '', ?)`, now)
	return err
}

// GetSettings returns the AI configuration.
func (s *Store) GetSettings(ctx context.Context) (*Settings, error) {
	var (
		item        Settings
		assistant   int
		mcp         int
		cipher      []byte
		updatedTime string
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT assistant_enabled, mcp_enabled, name, remark, model, endpoint_url, api_key_cipher, mcp_token, updated_time
		 FROM ai_settings WHERE singleton = 1`).Scan(
		&assistant, &mcp, &item.Name, &item.Remark, &item.Model, &item.EndpointURL, &cipher, &item.MCPToken, &updatedTime)
	if err != nil {
		return nil, err
	}
	item.AssistantEnabled = assistant == 1
	item.MCPEnabled = mcp == 1
	item.APIKey, err = s.codec.decryptString(cipher)
	if err != nil {
		return nil, err
	}
	item.UpdatedTime, err = parseTime(updatedTime)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// UpdateSettings updates the AI configuration.
func (s *Store) UpdateSettings(ctx context.Context, item *Settings) error {
	if item == nil {
		return fmt.Errorf("ai settings is nil")
	}
	now := time.Now().UTC()
	item.UpdatedTime = now
	cipherText, err := s.codec.encryptString(item.APIKey)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE ai_settings
		 SET assistant_enabled = ?, mcp_enabled = ?, name = ?, remark = ?, model = ?, endpoint_url = ?,
		     api_key_cipher = ?, mcp_token = ?, updated_time = ?
		 WHERE singleton = 1`,
		boolInt(item.AssistantEnabled), boolInt(item.MCPEnabled), item.Name, item.Remark, item.Model,
		item.EndpointURL, cipherText, item.MCPToken, formatTime(now))
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

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, value)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
