package pushmgr

import (
	"context"
	"database/sql"
	"fmt"
)

const pushSchemaSQL = `
CREATE TABLE IF NOT EXISTS push_connectors (
    id           TEXT PRIMARY KEY,
    type         TEXT NOT NULL,
    name         TEXT NOT NULL,
    push_config  TEXT NOT NULL DEFAULT '{}',
    run_state    TEXT NOT NULL DEFAULT 'running',
    version      INTEGER NOT NULL DEFAULT 1,
    created_time TEXT NOT NULL,
    updated_time TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_push_connectors_name ON push_connectors (name);
CREATE TABLE IF NOT EXISTS push_groups (
    id               TEXT PRIMARY KEY,
    name             TEXT NOT NULL,
    connector_id     TEXT NOT NULL,
    data_topic       TEXT NOT NULL DEFAULT '',
    command_topic    TEXT NOT NULL DEFAULT '',
    run_state        TEXT NOT NULL DEFAULT 'running',
    enable           INTEGER NOT NULL DEFAULT 1,
    push_interval_ms INTEGER NOT NULL,
    all_collections  INTEGER NOT NULL DEFAULT 0,
    max_collections_per_push_message INTEGER NOT NULL DEFAULT 10,
    version          INTEGER NOT NULL DEFAULT 1,
    created_time     TEXT NOT NULL,
    updated_time     TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_push_groups_connector ON push_groups (connector_id);
CREATE TABLE IF NOT EXISTS push_group_collections (
    push_group_id TEXT NOT NULL,
    collection_id TEXT NOT NULL,
    PRIMARY KEY (push_group_id, collection_id),
    FOREIGN KEY (push_group_id) REFERENCES push_groups(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_push_group_collections_collection ON push_group_collections (collection_id);
CREATE TABLE IF NOT EXISTS ready_send (
    seq           INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id      TEXT NOT NULL,
    connector_id  TEXT NOT NULL,
    topic         TEXT NOT NULL,
    payload       BLOB NOT NULL,
    payload_size  INTEGER NOT NULL,
    created_time  TEXT NOT NULL,
    FOREIGN KEY (group_id) REFERENCES push_groups(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ready_send_group_seq ON ready_send (group_id, seq);
CREATE INDEX IF NOT EXISTS idx_ready_send_connector ON ready_send (connector_id);
CREATE INDEX IF NOT EXISTS idx_ready_send_created ON ready_send (created_time);
`

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) Initialize(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("push store database is nil")
	}
	if _, err := s.db.ExecContext(ctx, pushSchemaSQL); err != nil {
		return err
	}
	columns := []struct {
		table string
		name  string
		ddl   string
	}{
		{"push_groups", "data_topic", "ALTER TABLE push_groups ADD COLUMN data_topic TEXT NOT NULL DEFAULT ''"},
		{"push_groups", "command_topic", "ALTER TABLE push_groups ADD COLUMN command_topic TEXT NOT NULL DEFAULT ''"},
		{"push_groups", "enable", "ALTER TABLE push_groups ADD COLUMN enable INTEGER NOT NULL DEFAULT 1"},
		{"push_groups", "run_state", "ALTER TABLE push_groups ADD COLUMN run_state TEXT NOT NULL DEFAULT ''"},
		{"push_connectors", "run_state", "ALTER TABLE push_connectors ADD COLUMN run_state TEXT NOT NULL DEFAULT 'running'"},
	}
	for _, column := range columns {
		if err := s.ensureColumn(ctx, column.table, column.name, column.ddl); err != nil {
			return err
		}
	}
	if _, err := s.db.ExecContext(ctx,
		`UPDATE push_groups SET run_state = CASE WHEN enable = 1 THEN 'running' ELSE 'stopped' END WHERE run_state = ''`); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx,
		`UPDATE push_connectors SET run_state = 'running' WHERE run_state = ''`); err != nil {
		return err
	}
	return s.backfillGroupTopics(ctx)
}

func (s *Store) ensureColumn(ctx context.Context, table, column, ddl string) error {
	rows, err := s.db.QueryContext(ctx, "PRAGMA table_info("+table+")")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid        int
			name       string
			typ        string
			notNull    int
			defaultV   interface{}
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultV, &primaryKey); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, ddl)
	return err
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}
