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

func (s *Store) backfillGroupTopics(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id, push_config FROM push_connectors`)
	if err != nil {
		return err
	}
	topics := make(map[string]struct {
		data    string
		command string
	})
	for rows.Next() {
		var id, raw string
		if err := rows.Scan(&id, &raw); err != nil {
			_ = rows.Close()
			return err
		}
		var config struct {
			DataTopic    string `json:"dataTopic"`
			CommandTopic string `json:"commandTopic"`
		}
		if err := json.Unmarshal([]byte(raw), &config); err == nil {
			topics[id] = struct {
				data    string
				command string
			}{data: config.DataTopic, command: config.CommandTopic}
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	groupRows, err := s.db.QueryContext(ctx, `SELECT id, connector_id, data_topic, command_topic FROM push_groups`)
	if err != nil {
		return err
	}
	type update struct {
		id      string
		data    string
		command string
	}
	updates := make([]update, 0)
	for groupRows.Next() {
		var id, connectorID, dataTopic, commandTopic string
		if err := groupRows.Scan(&id, &connectorID, &dataTopic, &commandTopic); err != nil {
			_ = groupRows.Close()
			return err
		}
		legacy, ok := topics[connectorID]
		if !ok {
			continue
		}
		nextData, nextCommand := dataTopic, commandTopic
		if nextData == "" {
			nextData = legacy.data
		}
		if nextCommand == "" {
			nextCommand = legacy.command
		}
		if nextData == dataTopic && nextCommand == commandTopic {
			continue
		}
		updates = append(updates, update{id: id, data: nextData, command: nextCommand})
	}
	if err := groupRows.Err(); err != nil {
		_ = groupRows.Close()
		return err
	}
	if err := groupRows.Close(); err != nil {
		return err
	}
	for _, item := range updates {
		if _, err := s.db.ExecContext(ctx,
			`UPDATE push_groups SET data_topic = ?, command_topic = ? WHERE id = ?`,
			item.data, item.command, item.id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ListAllConnectors(ctx context.Context) ([]*ConnectorRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, type, name, push_config, run_state, version, created_time, updated_time
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

func (s *Store) ListConnectors(ctx context.Context, q apis.PageQuery, keyword, connectorType string) ([]*ConnectorRecord, int64, error) {
	clauses := make([]string, 0, 2)
	args := make([]interface{}, 0, 2)
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		clauses = append(clauses, "c.name LIKE ?")
		args = append(args, "%"+keyword+"%")
	}
	if connectorType = strings.ToLower(strings.TrimSpace(connectorType)); connectorType != "" {
		clauses = append(clauses, "c.type = ?")
		args = append(args, connectorType)
	}
	where := "1=1"
	if len(clauses) > 0 {
		where = strings.Join(clauses, " AND ")
	}
	var total int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM push_connectors c WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	queryArgs := append(append([]interface{}{}, args...), q.Limit(), q.Offset())
	rows, err := s.db.QueryContext(ctx,
		`SELECT c.id, c.type, c.name, c.push_config, c.run_state,
		        (SELECT COUNT(1) FROM push_groups g WHERE g.connector_id = c.id) AS group_count,
		        c.version, c.created_time, c.updated_time
		 FROM push_connectors c WHERE `+where+`
		 ORDER BY c.created_time ASC, c.id ASC LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]*ConnectorRecord, 0)
	for rows.Next() {
		item, err := scanConnectorWithGroupCount(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, rows.Err()
}

func (s *Store) GetConnector(ctx context.Context, id string) (*ConnectorRecord, error) {
	return scanConnector(s.db.QueryRowContext(ctx,
		`SELECT id, type, name, push_config, run_state, version, created_time, updated_time
		 FROM push_connectors WHERE id = ?`, id))
}

func (s *Store) CreateConnector(ctx context.Context, item *ConnectorRecord) error {
	if item == nil {
		return fmt.Errorf("push connector is nil")
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO push_connectors (id, type, name, push_config, run_state, version, created_time, updated_time)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.Type, item.Name, string(item.PushConfig), item.RunState, item.Version,
		item.CreatedTime.UTC().Format(time.RFC3339Nano), item.UpdatedTime.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) UpdateConnector(ctx context.Context, item *ConnectorRecord) error {
	if item == nil {
		return fmt.Errorf("push connector is nil")
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE push_connectors SET type = ?, name = ?, push_config = ?, run_state = ?, version = ?, updated_time = ? WHERE id = ?`,
		item.Type, item.Name, string(item.PushConfig), item.RunState, item.Version,
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
		`SELECT id, name, connector_id, data_topic, command_topic, run_state, push_interval_ms, all_collections, collection_ids, max_collections_per_push_message, version, created_time, updated_time
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

func (s *Store) ListGroups(ctx context.Context, q apis.PageQuery, keyword, connectorID string) ([]*PushGroup, int64, error) {
	clauses := make([]string, 0, 2)
	args := make([]interface{}, 0, 2)
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		clauses = append(clauses, "name LIKE ?")
		args = append(args, "%"+keyword+"%")
	}
	if connectorID = strings.TrimSpace(connectorID); connectorID != "" {
		clauses = append(clauses, "connector_id = ?")
		args = append(args, connectorID)
	}
	where := "1=1"
	if len(clauses) > 0 {
		where = strings.Join(clauses, " AND ")
	}
	var total int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM push_groups WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	queryArgs := append(append([]interface{}{}, args...), q.Limit(), q.Offset())
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, connector_id, data_topic, command_topic, run_state, push_interval_ms, all_collections, collection_ids, max_collections_per_push_message, version, created_time, updated_time
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
		`SELECT id, name, connector_id, data_topic, command_topic, run_state, push_interval_ms, all_collections, collection_ids, max_collections_per_push_message, version, created_time, updated_time
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
		`INSERT INTO push_groups (id, name, connector_id, data_topic, command_topic, run_state, push_interval_ms, all_collections, collection_ids, max_collections_per_push_message, version, created_time, updated_time)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.Name, item.ConnectorID, item.DataTopic, item.CommandTopic, item.RunState, item.PushIntervalMs, boolInt(item.AllCollections), string(ids),
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
		`UPDATE push_groups SET name = ?, connector_id = ?, data_topic = ?, command_topic = ?, run_state = ?, push_interval_ms = ?, all_collections = ?, collection_ids = ?, max_collections_per_push_message = ?, version = ?, updated_time = ? WHERE id = ?`,
		item.Name, item.ConnectorID, item.DataTopic, item.CommandTopic, item.RunState, item.PushIntervalMs, boolInt(item.AllCollections), string(ids),
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
	if err := row.Scan(&item.ID, &item.Type, &item.Name, &config, &item.RunState, &item.Version, &createdRaw, &updatedRaw); err != nil {
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

func scanConnectorWithGroupCount(row rowScanner) (*ConnectorRecord, error) {
	var (
		item       ConnectorRecord
		config     string
		createdRaw string
		updatedRaw string
	)
	if err := row.Scan(
		&item.ID, &item.Type, &item.Name, &config, &item.RunState, &item.GroupCount,
		&item.Version, &createdRaw, &updatedRaw,
	); err != nil {
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
	if err := row.Scan(&item.ID, &item.Name, &item.ConnectorID, &item.DataTopic, &item.CommandTopic, &item.RunState, &item.PushIntervalMs, &all, &idsRaw, &item.MaxCollectionsPerPushMessage, &item.Version, &createdRaw, &updatedRaw); err != nil {
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
