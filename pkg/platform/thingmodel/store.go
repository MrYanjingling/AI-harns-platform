package thingmodel

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"harns-platform/pkg/apis"
	"harns-platform/pkg/platform/thingmodel/tsl"
)

// Store is the SQL access layer of the thing model module.
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

const (
	thingModelColumns = `id, version, thing_model_key, name, description, features,
 created_user, updated_user, created_time, updated_time`
	propertyColumns = `id, version, thing_model_id, identifier, name, data_type, rw_flag,
 required, description, custom, sort, created_user, updated_user, created_time, updated_time`
	propertyColumnsAliased = `p.id, p.version, p.thing_model_id, p.identifier, p.name, p.data_type, p.rw_flag,
 p.required, p.description, p.custom, p.sort, p.created_user, p.updated_user, p.created_time, p.updated_time`
)

type rowScanner interface {
	Scan(dest ...interface{}) error
}

// CreateThingModel writes the model and its initial properties in one transaction.
func (s *Store) CreateThingModel(ctx context.Context, m *ThingModel, properties []*Property) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	features, err := marshalFeatures(m.Features)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO thing_models (`+thingModelColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		m.ID, m.Version, m.ThingModelKey, m.Name, m.Description, features,
		m.CreatedUser, m.UpdatedUser, m.CreatedTime, m.UpdatedTime); err != nil {
		return err
	}
	if err := insertProperties(ctx, tx, properties); err != nil {
		return err
	}
	return tx.Commit()
}

// GetThingModel loads a model together with its properties.
func (s *Store) GetThingModel(ctx context.Context, id string) (*ThingModel, error) {
	m, err := scanThingModel(s.db.QueryRowContext(ctx,
		`SELECT `+thingModelColumns+` FROM thing_models WHERE id = ?`, id))
	if err != nil {
		return nil, err
	}
	properties, err := s.ListProperties(ctx, id)
	if err != nil {
		return nil, err
	}
	m.Properties = properties
	m.PropertyCount = len(properties)
	return m, nil
}

// ListThingModels returns a page of models with their property counts.
func (s *Store) ListThingModels(ctx context.Context, q apis.PageQuery, keyword string) ([]*ThingModel, int64, error) {
	where, args := "1=1", []interface{}{}
	if keyword != "" {
		where += " AND (m.name LIKE ? OR m.thing_model_key LIKE ?)"
		kw := "%" + keyword + "%"
		args = append(args, kw, kw)
	}

	var total int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM thing_models m WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT m.id, m.version, m.thing_model_key, m.name, m.description,
		 m.created_user, m.updated_user, m.created_time, m.updated_time,
		 (SELECT COUNT(1) FROM thing_model_properties p WHERE p.thing_model_id = m.id) AS property_count
		 FROM thing_models m WHERE `+where+` ORDER BY m.created_time DESC LIMIT ? OFFSET ?`,
		append(args, q.Limit(), q.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]*ThingModel, 0, q.Limit())
	for rows.Next() {
		m, err := scanThingModelWithCount(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, m)
	}
	return list, total, rows.Err()
}

// UpdateThingModel updates name/description and bumps the version.
func (s *Store) UpdateThingModel(ctx context.Context, m *ThingModel) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE thing_models SET name=?, description=?, updated_user=?, updated_time=?, version=version+1 WHERE id=?`,
		m.Name, m.Description, m.UpdatedUser, m.UpdatedTime, m.ID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// DeleteThingModel deletes the model, its properties, and all of its things in
// one transaction. The returned refs are used by post-commit cache cleanup.
func (s *Store) DeleteThingModel(ctx context.Context, id string) (*DeleteThingModelResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var modelKey string
	if err := tx.QueryRowContext(ctx,
		`SELECT thing_model_key FROM thing_models WHERE id = ? FOR UPDATE`, id).Scan(&modelKey); err != nil {
		return nil, err
	}

	rows, err := tx.QueryContext(ctx,
		`SELECT id, token, mqtt_client_id FROM things WHERE thing_model_id = ? FOR UPDATE`, id)
	if err != nil {
		return nil, err
	}
	refs := make([]ThingRef, 0)
	for rows.Next() {
		var ref ThingRef
		if err := rows.Scan(&ref.ID, &ref.Token, &ref.MQTTClientID); err != nil {
			rows.Close()
			return nil, err
		}
		refs = append(refs, ref)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM things WHERE thing_model_id = ?`, id)
	if err != nil {
		return nil, err
	}
	deletedThings, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM thing_model_properties WHERE thing_model_id = ?`, id); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM thing_models WHERE id = ?`, id); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &DeleteThingModelResult{
		ModelKey:      modelKey,
		DeletedThings: deletedThings,
		ThingRefs:     refs,
	}, nil
}

// ExistsThingModelName reports whether the name is taken.
func (s *Store) ExistsThingModelName(ctx context.Context, name, excludeID string) (bool, error) {
	return s.exists(ctx, "thing_models", "name", name, excludeID)
}

// ExistsThingModelKey reports whether the key is taken.
func (s *Store) ExistsThingModelKey(ctx context.Context, key, excludeID string) (bool, error) {
	return s.exists(ctx, "thing_models", "thing_model_key", key, excludeID)
}

// ListProperties returns the properties of a model ordered for display.
func (s *Store) ListProperties(ctx context.Context, modelID string) ([]Property, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+propertyColumns+` FROM thing_model_properties
		 WHERE thing_model_id = ? ORDER BY sort ASC, created_time ASC, id ASC`, modelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Property, 0)
	for rows.Next() {
		p, err := scanProperty(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *p)
	}
	return list, rows.Err()
}

// ListPropertiesByModelKey returns the properties of a model by business key.
func (s *Store) ListPropertiesByModelKey(ctx context.Context, modelKey string) ([]Property, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+propertyColumnsAliased+` FROM thing_model_properties p
		 JOIN thing_models m ON m.id = p.thing_model_id
		 WHERE m.thing_model_key = ?
		 ORDER BY p.sort ASC, p.created_time ASC, p.id ASC`, modelKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Property, 0)
	for rows.Next() {
		p, err := scanProperty(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *p)
	}
	return list, rows.Err()
}

// GetThingModelKey returns the business key of a model.
func (s *Store) GetThingModelKey(ctx context.Context, id string) (string, error) {
	var key string
	err := s.db.QueryRowContext(ctx,
		`SELECT thing_model_key FROM thing_models WHERE id = ?`, id).Scan(&key)
	return key, err
}

// GetProperty loads a single property by model and identifier.
func (s *Store) GetProperty(ctx context.Context, modelID, identifier string) (*Property, error) {
	return scanProperty(s.db.QueryRowContext(ctx,
		`SELECT `+propertyColumns+` FROM thing_model_properties
		 WHERE thing_model_id = ? AND identifier = ?`, modelID, identifier))
}

// AppendProperties inserts properties and bumps the model version.
func (s *Store) AppendProperties(ctx context.Context, modelID string, properties []*Property, operator string, now time.Time) error {
	if len(properties) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := insertProperties(ctx, tx, properties); err != nil {
		return err
	}
	if err := bumpThingModelVersion(ctx, tx, modelID, operator, now); err != nil {
		return err
	}
	return tx.Commit()
}

// UpdateProperty updates a single property row and bumps the model version.
func (s *Store) UpdateProperty(ctx context.Context, p *Property, operator string, now time.Time) (int64, error) {
	dataType, err := json.Marshal(&p.DataType)
	if err != nil {
		return 0, err
	}
	custom, err := marshalCustom(p.Custom)
	if err != nil {
		return 0, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		`UPDATE thing_model_properties SET name=?, data_type=?, rw_flag=?, required=?, description=?,
		 custom=?, sort=?, updated_user=?, updated_time=?, version=version+1 WHERE id=?`,
		p.Name, dataType, p.RWFlag, boolToInt(p.Required), p.Description,
		custom, p.Sort, operator, now, p.ID)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if n > 0 {
		if err := bumpThingModelVersion(ctx, tx, p.ThingModelID, operator, now); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return n, nil
}

// DeleteProperties deletes properties by identifier and bumps the model version.
func (s *Store) DeleteProperties(ctx context.Context, modelID string, identifiers []string, operator string, now time.Time) (int64, error) {
	if len(identifiers) == 0 {
		return 0, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	args := make([]interface{}, 0, len(identifiers)+1)
	args = append(args, modelID)
	for _, identifier := range identifiers {
		args = append(args, identifier)
	}
	res, err := tx.ExecContext(ctx,
		`DELETE FROM thing_model_properties WHERE thing_model_id = ? AND identifier IN (`+placeholders(len(identifiers))+`)`,
		args...)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if n > 0 {
		if err := bumpThingModelVersion(ctx, tx, modelID, operator, now); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return n, nil
}

// ReplaceThingModel updates the model header, replaces all properties and bumps
// the version in one transaction.
func (s *Store) ReplaceThingModel(ctx context.Context, m *ThingModel, properties []*Property) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	features, err := marshalFeatures(m.Features)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE thing_models SET name=?, description=?, features=?, updated_user=?, updated_time=?, version=version+1 WHERE id=?`,
		m.Name, m.Description, features, m.UpdatedUser, m.UpdatedTime, m.ID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM thing_model_properties WHERE thing_model_id = ?`, m.ID); err != nil {
		return err
	}
	if err := insertProperties(ctx, tx, properties); err != nil {
		return err
	}
	return tx.Commit()
}

func marshalFeatures(features []tsl.Feature) ([]byte, error) {
	if len(features) == 0 {
		return nil, nil
	}
	return json.Marshal(features)
}

func insertProperties(ctx context.Context, tx *sql.Tx, properties []*Property) error {
	if len(properties) == 0 {
		return nil
	}
	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO thing_model_properties (`+propertyColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range properties {
		dataType, err := json.Marshal(&p.DataType)
		if err != nil {
			return err
		}
		custom, err := marshalCustom(p.Custom)
		if err != nil {
			return err
		}
		if _, err := stmt.ExecContext(ctx,
			p.ID, p.Version, p.ThingModelID, p.Identifier, p.Name, dataType, p.RWFlag,
			boolToInt(p.Required), p.Description, custom, p.Sort,
			p.CreatedUser, p.UpdatedUser, p.CreatedTime, p.UpdatedTime); err != nil {
			return err
		}
	}
	return nil
}

func bumpThingModelVersion(ctx context.Context, tx *sql.Tx, modelID, operator string, now time.Time) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE thing_models SET version=version+1, updated_user=?, updated_time=? WHERE id=?`,
		operator, now, modelID)
	return err
}

func (s *Store) exists(ctx context.Context, table, column, value, excludeID string) (bool, error) {
	q, args := "SELECT COUNT(1) FROM "+table+" WHERE "+column+" = ?", []interface{}{value}
	if excludeID != "" {
		q += " AND id <> ?"
		args = append(args, excludeID)
	}
	var n int
	if err := s.db.QueryRowContext(ctx, q, args...).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

func scanThingModel(scanner rowScanner) (*ThingModel, error) {
	m := &ThingModel{}
	var features []byte
	if err := scanner.Scan(
		&m.ID, &m.Version, &m.ThingModelKey, &m.Name, &m.Description, &features,
		&m.CreatedUser, &m.UpdatedUser, &m.CreatedTime, &m.UpdatedTime); err != nil {
		return nil, err
	}
	if len(features) > 0 {
		if err := json.Unmarshal(features, &m.Features); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func scanThingModelWithCount(scanner rowScanner) (*ThingModel, error) {
	m := &ThingModel{}
	if err := scanner.Scan(
		&m.ID, &m.Version, &m.ThingModelKey, &m.Name, &m.Description,
		&m.CreatedUser, &m.UpdatedUser, &m.CreatedTime, &m.UpdatedTime, &m.PropertyCount); err != nil {
		return nil, err
	}
	return m, nil
}

func scanProperty(scanner rowScanner) (*Property, error) {
	p := &Property{}
	var (
		dataType []byte
		required int
		custom   []byte
	)
	if err := scanner.Scan(
		&p.ID, &p.Version, &p.ThingModelID, &p.Identifier, &p.Name, &dataType,
		&p.RWFlag, &required, &p.Description, &custom, &p.Sort,
		&p.CreatedUser, &p.UpdatedUser, &p.CreatedTime, &p.UpdatedTime); err != nil {
		return nil, err
	}
	p.Required = required != 0
	if len(dataType) > 0 {
		if err := json.Unmarshal(dataType, &p.DataType); err != nil {
			return nil, err
		}
	}
	if len(custom) > 0 {
		if err := json.Unmarshal(custom, &p.Custom); err != nil {
			return nil, err
		}
	}
	return p, nil
}

func marshalCustom(custom map[string]interface{}) (interface{}, error) {
	if custom == nil {
		return nil, nil
	}
	return json.Marshal(custom)
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	out := make([]byte, 0, n*2-1)
	for i := 0; i < n; i++ {
		if i > 0 {
			out = append(out, ',')
		}
		out = append(out, '?')
	}
	return string(out)
}
