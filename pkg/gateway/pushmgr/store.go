package pushmgr

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"harns-platform/pkg/apis"
)

const pushSchemaSQL = `
CREATE TABLE IF NOT EXISTS push_connectors (
    id           TEXT PRIMARY KEY,
    type         TEXT NOT NULL,
    name         TEXT NOT NULL,
    push_config  TEXT NOT NULL DEFAULT '{}',
    version      INTEGER NOT NULL DEFAULT 1,
    created_time TEXT NOT NULL,
    updated_time TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_push_connectors_name ON push_connectors (name);
CREATE TABLE IF NOT EXISTS push_groups (
    id               TEXT PRIMARY KEY,
    name             TEXT NOT NULL,
    connector_id     TEXT NOT NULL,
    push_interval_ms INTEGER NOT NULL,
    all_collections  INTEGER NOT NULL DEFAULT 0,
    collection_ids   TEXT NOT NULL DEFAULT '[]',
    max_collections_per_push_message INTEGER NOT NULL DEFAULT 10,
    version          INTEGER NOT NULL DEFAULT 1,
    created_time     TEXT NOT NULL,
    updated_time     TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_push_groups_connector ON push_groups (connector_id);
`

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) Initialize(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("push store database is nil")
	}
	_, err := s.db.ExecContext(ctx, pushSchemaSQL)
	return err
}

func (s *Store) ListAllConnectors(ctx context.Context) ([]*ConnectorRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, type, name, push_config, version, created_time, updated_time
		 FROM push_connectors ORDER BY created_time ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*ConnectorRecord, 0)
	for rows.Next() {
		item, err := scanConnector(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) ListConnectors(ctx context.Context, q apis.PageQuery, keyword string) ([]*ConnectorRecord, int64, error) {
	where := "1=1"
	args := make([]interface{}, 0)
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		where = "name LIKE ?"
		args = append(args, "%"+keyword+"%")
	}
	var total int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM push_connectors WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	queryArgs := append(append([]interface{}{}, args...), q.Limit(), q.Offset())
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, type, name, push_config, version, created_time, updated_time
		 FROM push_connectors WHERE `+where+` ORDER BY created_time ASC, id ASC LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]*ConnectorRecord, 0)
	for rows.Next() {
		item, err := scanConnector(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, rows.Err()
}

func (s *Store) GetConnector(ctx context.Context, id string) (*ConnectorRecord, error) {
	return scanConnector(s.db.QueryRowContext(ctx,
		`SELECT id, type, name, push_config, version, created_time, updated_time
		 FROM push_connectors WHERE id = ?`, id))
}

func (s *Store) CreateConnector(ctx context.Context, item *ConnectorRecord) error {
	if item == nil {
		return fmt.Errorf("push connector is nil")
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO push_connectors (id, type, name, push_config, version, created_time, updated_time)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.Type, item.Name, string(item.PushConfig), item.Version,
		item.CreatedTime.UTC().Format(time.RFC3339Nano), item.UpdatedTime.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) UpdateConnector(ctx context.Context, item *ConnectorRecord) error {
	if item == nil {
		return fmt.Errorf("push connector is nil")
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE push_connectors SET type = ?, name = ?, push_config = ?, version = ?, updated_time = ? WHERE id = ?`,
		item.Type, item.Name, string(item.PushConfig), item.Version,
		item.UpdatedTime.UTC().Format(time.RFC3339Nano), item.ID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrConnectorNotFound
	}
	return nil
}

func (s *Store) DeleteConnector(ctx context.Context, id string, cascade bool) ([]string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM push_groups WHERE connector_id = ?`, id).Scan(&count); err != nil {
		return nil, err
	}
	if count > 0 && !cascade {
		return nil, ErrConnectorInUse
	}
	groupIDs := make([]string, 0)
	if cascade && count > 0 {
		rows, err := tx.QueryContext(ctx, `SELECT id FROM push_groups WHERE connector_id = ?`, id)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var groupID string
			if err := rows.Scan(&groupID); err != nil {
				_ = rows.Close()
				return nil, err
			}
			groupIDs = append(groupIDs, groupID)
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM push_groups WHERE connector_id = ?`, id); err != nil {
			return nil, err
		}
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM push_connectors WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, ErrConnectorNotFound
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return groupIDs, nil
}

func (s *Store) ListAllGroups(ctx context.Context) ([]*PushGroup, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, connector_id, push_interval_ms, all_collections, collection_ids, max_collections_per_push_message, version, created_time, updated_time
		 FROM push_groups ORDER BY created_time ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*PushGroup, 0)
	for rows.Next() {
		item, err := scanGroup(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) ListGroups(ctx context.Context, q apis.PageQuery, keyword string) ([]*PushGroup, int64, error) {
	where := "1=1"
	args := make([]interface{}, 0)
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		where = "name LIKE ?"
		args = append(args, "%"+keyword+"%")
	}
	var total int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM push_groups WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	queryArgs := append(append([]interface{}{}, args...), q.Limit(), q.Offset())
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, connector_id, push_interval_ms, all_collections, collection_ids, max_collections_per_push_message, version, created_time, updated_time
		 FROM push_groups WHERE `+where+` ORDER BY created_time ASC, id ASC LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]*PushGroup, 0)
	for rows.Next() {
		item, err := scanGroup(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, rows.Err()
}

func (s *Store) GetGroup(ctx context.Context, id string) (*PushGroup, error) {
	return scanGroup(s.db.QueryRowContext(ctx,
		`SELECT id, name, connector_id, push_interval_ms, all_collections, collection_ids, max_collections_per_push_message, version, created_time, updated_time
		 FROM push_groups WHERE id = ?`, id))
}

func (s *Store) CreateGroup(ctx context.Context, item *PushGroup) error {
	if item == nil {
		return fmt.Errorf("push group is nil")
	}
	ids, err := json.Marshal(item.CollectionIDs)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO push_groups (id, name, connector_id, push_interval_ms, all_collections, collection_ids, max_collections_per_push_message, version, created_time, updated_time)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.Name, item.ConnectorID, item.PushIntervalMs, boolInt(item.AllCollections), string(ids),
		item.MaxCollectionsPerPushMessage, item.Version, item.CreatedTime.UTC().Format(time.RFC3339Nano), item.UpdatedTime.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) UpdateGroup(ctx context.Context, item *PushGroup) error {
	if item == nil {
		return fmt.Errorf("push group is nil")
	}
	ids, err := json.Marshal(item.CollectionIDs)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE push_groups SET name = ?, connector_id = ?, push_interval_ms = ?, all_collections = ?, collection_ids = ?, max_collections_per_push_message = ?, version = ?, updated_time = ? WHERE id = ?`,
		item.Name, item.ConnectorID, item.PushIntervalMs, boolInt(item.AllCollections), string(ids),
		item.MaxCollectionsPerPushMessage, item.Version, item.UpdatedTime.UTC().Format(time.RFC3339Nano), item.ID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrGroupNotFound
	}
	return nil
}

func (s *Store) DeleteGroup(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM push_groups WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrGroupNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanConnector(row rowScanner) (*ConnectorRecord, error) {
	var (
		item       ConnectorRecord
		config     string
		createdRaw string
		updatedRaw string
	)
	if err := row.Scan(&item.ID, &item.Type, &item.Name, &config, &item.Version, &createdRaw, &updatedRaw); err != nil {
		return nil, err
	}
	item.PushConfig = json.RawMessage(config)
	created, err := time.Parse(time.RFC3339Nano, createdRaw)
	if err != nil {
		return nil, err
	}
	updated, err := time.Parse(time.RFC3339Nano, updatedRaw)
	if err != nil {
		return nil, err
	}
	item.CreatedTime = created
	item.UpdatedTime = updated
	return &item, nil
}

func scanGroup(row rowScanner) (*PushGroup, error) {
	var (
		item       PushGroup
		all        int
		idsRaw     string
		createdRaw string
		updatedRaw string
	)
	if err := row.Scan(&item.ID, &item.Name, &item.ConnectorID, &item.PushIntervalMs, &all, &idsRaw, &item.MaxCollectionsPerPushMessage, &item.Version, &createdRaw, &updatedRaw); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(idsRaw), &item.CollectionIDs); err != nil {
		return nil, err
	}
	item.AllCollections = all != 0
	created, err := time.Parse(time.RFC3339Nano, createdRaw)
	if err != nil {
		return nil, err
	}
	updated, err := time.Parse(time.RFC3339Nano, updatedRaw)
	if err != nil {
		return nil, err
	}
	item.CreatedTime = created
	item.UpdatedTime = updated
	return &item, nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
