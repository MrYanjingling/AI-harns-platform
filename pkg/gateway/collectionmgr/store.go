package collectionmgr

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"harns-platform/pkg/apis"
	"harns-platform/pkg/gateway/collection"
)

var (
	ErrCollectionNotFound = errors.New("collection not found")
	ErrVariableConflict   = errors.New("variable conflict")
)

// Store persists collections and their variables in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a collection store.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

const collectionSchemaSQL = `
CREATE TABLE IF NOT EXISTS collections (
    id                 TEXT PRIMARY KEY,
    name               TEXT NOT NULL,
    version            TEXT NOT NULL,
    driver_type        TEXT NOT NULL,
    driver_model       TEXT NOT NULL,
    collect_status     TEXT NOT NULL DEFAULT 'stopped',
    collector_cycle_ms INTEGER NOT NULL,
    protocol_config    TEXT NOT NULL DEFAULT '{}',
    created_time       TEXT NOT NULL,
    updated_time       TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_collections_driver ON collections (driver_type, driver_model);
CREATE INDEX IF NOT EXISTS idx_collections_name ON collections (name);
CREATE TABLE IF NOT EXISTS collection_variables (
    collection_id TEXT NOT NULL,
    name          TEXT NOT NULL,
    variable_type TEXT NOT NULL,
    data_type     TEXT NOT NULL,
    address       TEXT NOT NULL DEFAULT '',
    access_mode   TEXT NOT NULL,
    is_calculated INTEGER NOT NULL DEFAULT 0,
    expression    TEXT NOT NULL DEFAULT '',
    value_offset  REAL NOT NULL DEFAULT 0,
    rate          REAL NOT NULL DEFAULT 1,
    precision     INTEGER NOT NULL DEFAULT 0,
    sort          INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (collection_id, name),
    FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE CASCADE,
    CHECK (access_mode IN ('r', 'w', 'rw')),
    CHECK (is_calculated IN (0, 1)),
    CHECK (
        (is_calculated = 0 AND address <> '' AND expression = '')
        OR
        (is_calculated = 1 AND address = '' AND expression <> '' AND access_mode = 'r')
    )
);
CREATE INDEX IF NOT EXISTS idx_collection_variables_sort ON collection_variables (collection_id, sort);
`

// Initialize creates the collection tables.
func (s *Store) Initialize(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, collectionSchemaSQL)
	return err
}

// DriverExists reports whether a driver type/model pair exists.
func (s *Store) DriverExists(ctx context.Context, driverType, driverModel string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM drivers WHERE driver_type = ? AND model = ?`,
		driverType, driverModel).Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// DriverSchema returns the schema JSON for a driver type/model pair.
func (s *Store) DriverSchema(ctx context.Context, driverType, driverModel string) (json.RawMessage, error) {
	var schema string
	err := s.db.QueryRowContext(ctx,
		`SELECT schema FROM drivers WHERE driver_type = ? AND model = ?`,
		driverType, driverModel).Scan(&schema)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(schema), nil
}

// Create inserts a collection and its variables in one transaction.
func (s *Store) Create(ctx context.Context, record *CollectionRecord) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO collections (
			id, name, version, driver_type, driver_model, collect_status,
			collector_cycle_ms, protocol_config, created_time, updated_time
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.ID, record.Name, record.Version, record.DriverType, record.DriverModel,
		record.CollectStatus, record.CollectorCycleMs, string(record.ProtocolConfig),
		record.CreatedTime.UTC().Format(time.RFC3339Nano),
		record.UpdatedTime.UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO collection_variables (
			collection_id, name, variable_type, data_type, address, access_mode,
			is_calculated, expression, value_offset, rate, precision, sort
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, variable := range record.Variables {
		if _, err := stmt.ExecContext(ctx,
			record.ID, variable.Name, variable.Type, variable.DataType, variable.Address,
			string(variable.AccessMode), boolInt(variable.IsCalculated), variable.Expression,
			variable.Offset, variable.Rate, variable.Precision, i); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetCollection loads one collection without variables.
func (s *Store) GetCollection(ctx context.Context, id string) (*CollectionRecord, error) {
	var (
		record         CollectionRecord
		protocolConfig string
		createdTime    string
		updatedTime    string
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, version, driver_type, driver_model, collect_status,
			collector_cycle_ms, protocol_config, created_time, updated_time
		 FROM collections WHERE id = ?`, id).Scan(
		&record.ID, &record.Name, &record.Version, &record.DriverType, &record.DriverModel,
		&record.CollectStatus, &record.CollectorCycleMs, &protocolConfig, &createdTime, &updatedTime)
	if err != nil {
		return nil, err
	}
	record.ProtocolConfig = json.RawMessage(protocolConfig)
	record.CreatedTime, err = time.Parse(time.RFC3339Nano, createdTime)
	if err != nil {
		return nil, err
	}
	record.UpdatedTime, err = time.Parse(time.RFC3339Nano, updatedTime)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// ListAll returns all collections with their variables.
func (s *Store) ListAll(ctx context.Context) ([]*CollectionRecord, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM collections ORDER BY created_time ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	records := make([]*CollectionRecord, 0, len(ids))
	for _, id := range ids {
		record, err := s.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

// List returns a page of collections without variables.
func (s *Store) List(ctx context.Context, q apis.PageQuery, keyword, driverType, driverModel string) ([]*CollectionRecord, int64, error) {
	where := "1=1"
	args := make([]interface{}, 0, 4)
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		where += ` AND name LIKE ? ESCAPE '\'`
		args = append(args, fuzzyPattern(keyword))
	}
	if driverType = strings.TrimSpace(driverType); driverType != "" {
		where += " AND driver_type = ?"
		args = append(args, driverType)
	}
	if driverModel = strings.TrimSpace(driverModel); driverModel != "" {
		where += " AND driver_model = ?"
		args = append(args, driverModel)
	}

	var total int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM collections WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	queryArgs := append(append([]interface{}{}, args...), q.Limit(), q.Offset())
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, version, driver_type, driver_model, collect_status,
			collector_cycle_ms, protocol_config, created_time, updated_time
		 FROM collections WHERE `+where+` ORDER BY created_time DESC, id ASC LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]*CollectionRecord, 0)
	for rows.Next() {
		record, err := scanCollection(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, record)
	}
	return list, total, rows.Err()
}

// Update updates collection metadata and protocol config without touching variables.
func (s *Store) Update(ctx context.Context, record *CollectionRecord) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE collections SET name = ?, version = ?, collect_status = ?, collector_cycle_ms = ?,
			protocol_config = ?, updated_time = ? WHERE id = ?`,
		record.Name, record.Version, record.CollectStatus, record.CollectorCycleMs,
		string(record.ProtocolConfig), record.UpdatedTime.UTC().Format(time.RFC3339Nano), record.ID)
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

// Delete deletes one collection; its variables cascade.
func (s *Store) Delete(ctx context.Context, id string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM collections WHERE id = ?`, id)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ListVariables returns a page of variables for one collection.
func (s *Store) ListVariables(ctx context.Context, collectionID string, q apis.PageQuery, keyword string) ([]*collection.Variable, int64, error) {
	where := "collection_id = ?"
	args := []interface{}{collectionID}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		where += ` AND name LIKE ? ESCAPE '\'`
		args = append(args, fuzzyPattern(keyword))
	}
	var total int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM collection_variables WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	queryArgs := append(append([]interface{}{}, args...), q.Limit(), q.Offset())
	rows, err := s.db.QueryContext(ctx,
		`SELECT name, variable_type, data_type, address, access_mode,
			is_calculated, expression, value_offset, rate, precision
		 FROM collection_variables WHERE `+where+`
		 ORDER BY sort ASC, name ASC LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	variables := make([]*collection.Variable, 0)
	for rows.Next() {
		variable, err := scanVariable(rows)
		if err != nil {
			return nil, 0, err
		}
		variables = append(variables, variable)
	}
	return variables, total, rows.Err()
}

// AddVariables appends variables to one collection.
func (s *Store) AddVariables(ctx context.Context, collectionID string, variables []*collection.Variable) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var maxSort int
	if err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(sort), 0) FROM collection_variables WHERE collection_id = ?`,
		collectionID).Scan(&maxSort); err != nil {
		return 0, err
	}

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO collection_variables (
			collection_id, name, variable_type, data_type, address, access_mode,
			is_calculated, expression, value_offset, rate, precision, sort
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	for i, variable := range variables {
		if _, err := stmt.ExecContext(ctx,
			collectionID, variable.Name, variable.Type, variable.DataType, variable.Address,
			string(variable.AccessMode), boolInt(variable.IsCalculated), variable.Expression,
			variable.Offset, variable.Rate, variable.Precision, maxSort+i+1); err != nil {
			return 0, mapCollectionWriteError(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return int64(len(variables)), nil
}

// DeleteVariables deletes variables by name from one collection.
func (s *Store) DeleteVariables(ctx context.Context, collectionID string, names []string) (int64, error) {
	if len(names) == 0 {
		return 0, nil
	}
	args := make([]interface{}, 0, len(names)+1)
	args = append(args, collectionID)
	for _, name := range names {
		args = append(args, name)
	}
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM collection_variables WHERE collection_id = ? AND name IN (`+placeholders(len(names))+`)`, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Get loads one collection and its variables.
func (s *Store) Get(ctx context.Context, id string) (*CollectionRecord, error) {
	var (
		record         CollectionRecord
		protocolConfig string
		createdTime    string
		updatedTime    string
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, version, driver_type, driver_model, collect_status,
			collector_cycle_ms, protocol_config, created_time, updated_time
		 FROM collections WHERE id = ?`, id).Scan(
		&record.ID, &record.Name, &record.Version, &record.DriverType, &record.DriverModel,
		&record.CollectStatus, &record.CollectorCycleMs, &protocolConfig, &createdTime, &updatedTime)
	if err != nil {
		return nil, err
	}
	record.ProtocolConfig = json.RawMessage(protocolConfig)
	record.CreatedTime, err = time.Parse(time.RFC3339Nano, createdTime)
	if err != nil {
		return nil, err
	}
	record.UpdatedTime, err = time.Parse(time.RFC3339Nano, updatedTime)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT name, variable_type, data_type, address, access_mode,
			is_calculated, expression, value_offset, rate, precision
		 FROM collection_variables WHERE collection_id = ? ORDER BY sort ASC, name ASC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		variable, err := scanVariable(rows)
		if err != nil {
			return nil, err
		}
		record.Variables = append(record.Variables, variable)
	}
	return &record, rows.Err()
}

func scanCollection(scanner rowScanner) (*CollectionRecord, error) {
	var (
		record         CollectionRecord
		protocolConfig string
		createdTime    string
		updatedTime    string
	)
	if err := scanner.Scan(
		&record.ID, &record.Name, &record.Version, &record.DriverType, &record.DriverModel,
		&record.CollectStatus, &record.CollectorCycleMs, &protocolConfig, &createdTime, &updatedTime,
	); err != nil {
		return nil, err
	}
	record.ProtocolConfig = json.RawMessage(protocolConfig)
	var err error
	record.CreatedTime, err = time.Parse(time.RFC3339Nano, createdTime)
	if err != nil {
		return nil, err
	}
	record.UpdatedTime, err = time.Parse(time.RFC3339Nano, updatedTime)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func scanVariable(scanner rowScanner) (*collection.Variable, error) {
	var (
		variable     collection.Variable
		accessMode   string
		isCalculated int
	)
	if err := scanner.Scan(
		&variable.Name, &variable.Type, &variable.DataType, &variable.Address, &accessMode,
		&isCalculated, &variable.Expression, &variable.Offset, &variable.Rate, &variable.Precision,
	); err != nil {
		return nil, err
	}
	variable.AccessMode = collection.AccessMode(accessMode)
	variable.IsCalculated = isCalculated == 1
	return &variable, nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func fuzzyPattern(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return "%" + value + "%"
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", n), ",")
}

func mapCollectionWriteError(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "unique constraint failed") {
		return ErrVariableConflict
	}
	return err
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
