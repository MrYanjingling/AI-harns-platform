package drivermgr

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrConflict        = errors.New("driver conflict")
	ErrBuiltinReadOnly = errors.New("builtin driver is read-only")
)

const createTableSQL = `
CREATE TABLE IF NOT EXISTS drivers (
    id              TEXT PRIMARY KEY,
    driver_type   TEXT NOT NULL,
    model           TEXT NOT NULL,
    builtin_driver  INTEGER NOT NULL DEFAULT 0,
    schema          TEXT NOT NULL DEFAULT '{}',
    plugin_name     TEXT NOT NULL,
    created_time    TEXT NOT NULL,
    UNIQUE (driver_type, model),
    UNIQUE (plugin_name),
    CHECK (builtin_driver IN (0, 1)),
    CHECK (plugin_name <> '')
);
CREATE INDEX IF NOT EXISTS idx_drivers_driver_type ON drivers (driver_type);
`

// Store is the SQLite persistence layer for driver metadata.
type Store struct {
	db *sql.DB
}

// NewStore creates a driver metadata store.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Initialize creates the schema and ensures built-in driver rows exist.
func (s *Store) Initialize(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, createTableSQL); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, builtin := range BuiltinDrivers {
		if _, err := ParseSchema(json.RawMessage(builtin.Schema)); err != nil {
			return fmt.Errorf("invalid builtin driver schema %s: %w", builtin.PluginName, err)
		}
		var builtinDriver int
		err := tx.QueryRowContext(ctx,
			`SELECT builtin_driver FROM drivers WHERE plugin_name = ?`, builtin.PluginName).Scan(&builtinDriver)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO drivers (id, driver_type, model, builtin_driver, schema, plugin_name, created_time)
				 VALUES (?, ?, ?, 1, ?, ?, ?)`,
				builtin.ID, builtin.DriverType, builtin.Model, builtin.Schema, builtin.PluginName, now); err != nil {
				return err
			}
		case err != nil:
			return err
		case builtinDriver == 0:
			return ErrConflict
		default:
			if _, err := tx.ExecContext(ctx,
				`UPDATE drivers SET driver_type = ?, model = ?, schema = ? WHERE plugin_name = ? AND builtin_driver = 1`,
				builtin.DriverType, builtin.Model, builtin.Schema, builtin.PluginName); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// List returns all drivers, optionally filtered by driver type.
func (s *Store) List(ctx context.Context, driverType string) ([]*Driver, error) {
	query := `SELECT id, driver_type, model, builtin_driver, schema, plugin_name, created_time FROM drivers`
	args := make([]interface{}, 0, 1)
	if driverType = strings.TrimSpace(driverType); driverType != "" {
		query += ` WHERE driver_type LIKE ? ESCAPE '\'`
		args = append(args, fuzzyPattern(driverType))
	}
	query += ` ORDER BY driver_type ASC, model ASC, id ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*Driver, 0)
	for rows.Next() {
		item, err := scanDriver(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// Get returns one driver by ID.
func (s *Store) Get(ctx context.Context, id string) (*Driver, error) {
	return scanDriver(s.db.QueryRowContext(ctx,
		`SELECT id, driver_type, model, builtin_driver, schema, plugin_name, created_time
		 FROM drivers WHERE id = ?`, id))
}

// GetByModel returns one driver by model.
func (s *Store) GetByModel(ctx context.Context, model string) (*Driver, error) {
	return scanDriver(s.db.QueryRowContext(ctx,
		`SELECT id, driver_type, model, builtin_driver, schema, plugin_name, created_time
		 FROM drivers WHERE model = ? ORDER BY builtin_driver DESC, id ASC LIMIT 1`, model))
}

// Create inserts a custom driver.
func (s *Store) Create(ctx context.Context, item *Driver) error {
	schema, err := schemaText(item.Schema)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO drivers (id, driver_type, model, builtin_driver, schema, plugin_name, created_time)
		 VALUES (?, ?, ?, 0, ?, ?, ?)`,
		item.ID, item.DriverType, item.Model, schema, item.PluginName, item.CreatedTime.UTC().Format(time.RFC3339Nano))
	return mapWriteError(err)
}

// Update updates a custom driver.
func (s *Store) Update(ctx context.Context, item *Driver) error {
	schema, err := schemaText(item.Schema)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE drivers SET driver_type = ?, model = ?, schema = ?, plugin_name = ?
		 WHERE id = ? AND builtin_driver = 0`,
		item.DriverType, item.Model, schema, item.PluginName, item.ID)
	if err != nil {
		return mapWriteError(err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrBuiltinReadOnly
	}
	return nil
}

// Delete removes a custom driver.
func (s *Store) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM drivers WHERE id = ? AND builtin_driver = 0`, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrBuiltinReadOnly
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanDriver(scanner rowScanner) (*Driver, error) {
	var (
		item          Driver
		builtinDriver int
		schema        string
		createdTime   string
	)
	if err := scanner.Scan(
		&item.ID, &item.DriverType, &item.Model, &builtinDriver, &schema, &item.PluginName, &createdTime,
	); err != nil {
		return nil, err
	}
	item.BuiltinDriver = builtinDriver == 1
	item.Schema = json.RawMessage(schema)
	parsed, err := time.Parse(time.RFC3339Nano, createdTime)
	if err != nil {
		return nil, err
	}
	item.CreatedTime = parsed
	return &item, nil
}

func schemaText(schema json.RawMessage) (string, error) {
	if len(schema) == 0 {
		return "{}", nil
	}
	if !json.Valid(schema) {
		return "", ErrConflict
	}
	return string(schema), nil
}

func fuzzyPattern(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return "%" + value + "%"
}

func mapWriteError(err error) error {
	if err == nil {
		return nil
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "unique constraint failed") {
		return ErrConflict
	}
	return err
}
