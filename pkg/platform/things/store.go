package things

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"harns-platform/pkg/apis"
)

// Store is the SQL access layer of the things module.
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

const thingColumns = `id, version, thing_model_id, device_name, device_key, device_secret, token,
 up_topic, down_topic, device_desc, features, image, mqtt_client_id, mqtt_username, mqtt_password, enable,
 created_user, updated_user, created_time, updated_time`

const authThingColumns = `t.id, t.version, t.thing_model_id, t.device_name, t.device_key, t.device_secret,
 t.token, t.up_topic, t.down_topic, t.device_desc, t.mqtt_client_id, t.mqtt_username,
 t.mqtt_password, t.enable, t.created_user, t.updated_user, t.created_time, t.updated_time,
 m.thing_model_key`

// CreateThings inserts things in one transaction.
func (s *Store) CreateThings(ctx context.Context, items []*Thing) error {
	if len(items) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO things (`+thingColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, item := range items {
		if _, err := stmt.ExecContext(ctx,
			item.ID, item.Version, item.ThingModelID, item.DeviceName, item.DeviceKey,
			item.DeviceSecret, item.Token, item.UpTopic, item.DownTopic, item.DeviceDesc,
			marshalFeatureValues(item.Features), item.Image,
			item.MQTTClientID, item.MQTTUsername, item.MQTTPassword, item.Enable,
			item.CreatedUser, item.UpdatedUser, item.CreatedTime, item.UpdatedTime); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetThing loads a thing by id.
func (s *Store) GetThing(ctx context.Context, id string) (*Thing, error) {
	return scanThing(s.db.QueryRowContext(ctx, `SELECT `+thingColumns+` FROM things WHERE id = ?`, id))
}

// AuthenticateThing loads an enabled thing matching the MQTT credentials.
func (s *Store) AuthenticateThing(ctx context.Context, clientID, username, password string) (*AuthThing, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+authThingColumns+`
		 FROM things t JOIN thing_models m ON m.id = t.thing_model_id
		 WHERE t.mqtt_client_id = ? AND t.mqtt_username = ? AND t.mqtt_password = ? AND t.enable = 1
		 LIMIT 1`, clientID, username, password)

	item := &AuthThing{}
	if err := row.Scan(
		&item.ID, &item.Version, &item.ThingModelID, &item.DeviceName, &item.DeviceKey,
		&item.DeviceSecret, &item.Token, &item.UpTopic, &item.DownTopic, &item.DeviceDesc,
		&item.MQTTClientID, &item.MQTTUsername, &item.MQTTPassword, &item.Enable,
		&item.CreatedUser, &item.UpdatedUser, &item.CreatedTime, &item.UpdatedTime,
		&item.ThingModelKey); err != nil {
		return nil, err
	}
	return item, nil
}

// GetThingModelFeatureIdentifiers returns the feature identifiers defined by a model.
func (s *Store) GetThingModelFeatureIdentifiers(ctx context.Context, modelID string) (map[string]struct{}, error) {
	var raw []byte
	if err := s.db.QueryRowContext(ctx, `SELECT features FROM thing_models WHERE id = ?`, modelID).Scan(&raw); err != nil {
		return nil, err
	}
	out := make(map[string]struct{})
	if len(raw) == 0 {
		return out, nil
	}
	var features []struct {
		Identifier string `json:"identifier"`
	}
	if err := json.Unmarshal(raw, &features); err != nil {
		return nil, err
	}
	for _, feature := range features {
		if feature.Identifier != "" {
			out[feature.Identifier] = struct{}{}
		}
	}
	return out, nil
}

// GetThingByToken loads an enabled thing identity by HTTP token.
func (s *Store) GetThingByToken(ctx context.Context, token string) (*ThingIdentity, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT t.id, t.thing_model_id, m.thing_model_key, t.device_key, t.device_name,
		 t.mqtt_client_id, t.mqtt_username, t.up_topic, t.down_topic, t.enable
		 FROM things t JOIN thing_models m ON m.id = t.thing_model_id
		 WHERE t.token = ? AND t.enable = 1 LIMIT 1`, token)
	item := &ThingIdentity{}
	if err := row.Scan(
		&item.ID, &item.ThingModelID, &item.ThingModelKey, &item.DeviceKey, &item.DeviceName,
		&item.MQTTClientID, &item.MQTTUsername, &item.UpTopic, &item.DownTopic,
		&item.Enable); err != nil {
		return nil, err
	}
	return item, nil
}

// ListThingTokens returns the tokens of the given thing ids.
func (s *Store) ListThingTokens(ctx context.Context, ids []string) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	args := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT token FROM things WHERE id IN (`+placeholders(len(ids))+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tokens := make([]string, 0, len(ids))
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}

// ListThings returns a page of things filtered by keyword.
func (s *Store) ListThings(ctx context.Context, q apis.PageQuery, keyword string) ([]*Thing, int64, error) {
	where, args := "1=1", []interface{}{}
	if keyword != "" {
		where += " AND (device_name LIKE ? OR device_key LIKE ?)"
		kw := "%" + keyword + "%"
		args = append(args, kw, kw)
	}

	var total int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM things WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT `+thingColumns+` FROM things WHERE `+where+` ORDER BY created_time DESC LIMIT ? OFFSET ?`,
		append(args, q.Limit(), q.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]*Thing, 0, q.Limit())
	for rows.Next() {
		item, err := scanThing(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, item)
	}
	return list, total, rows.Err()
}

// UpdateThing updates the mutable fields and bumps the version.
func (s *Store) UpdateThing(ctx context.Context, item *Thing) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE things SET device_name=?, device_desc=?, features=?, image=?, mqtt_client_id=?, mqtt_username=?,
		 mqtt_password=?, enable=?, updated_user=?, updated_time=?, version=version+1 WHERE id=?`,
		item.DeviceName, item.DeviceDesc, marshalFeatureValues(item.Features), item.Image, item.MQTTClientID, item.MQTTUsername,
		item.MQTTPassword, item.Enable, item.UpdatedUser, item.UpdatedTime, item.ID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// DeleteThings deletes things by id.
func (s *Store) DeleteThings(ctx context.Context, ids []string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	args := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM things WHERE id IN (`+placeholders(len(ids))+`)`, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ThingModelExists reports whether a thing model exists.
func (s *Store) ThingModelExists(ctx context.Context, id string) (bool, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM thing_models WHERE id = ?`, id).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

// ExistsDeviceName reports whether the name is taken inside the model.
func (s *Store) ExistsDeviceName(ctx context.Context, modelID, name, excludeID string) (bool, error) {
	q, args := `SELECT COUNT(1) FROM things WHERE thing_model_id = ? AND device_name = ?`, []interface{}{modelID, name}
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

// ExistsDeviceKey reports whether the device key is taken.
func (s *Store) ExistsDeviceKey(ctx context.Context, key, excludeID string) (bool, error) {
	return s.exists(ctx, "device_key", key, excludeID)
}

// ExistsMQTTClientID reports whether the MQTT client id is taken.
func (s *Store) ExistsMQTTClientID(ctx context.Context, clientID, excludeID string) (bool, error) {
	return s.exists(ctx, "mqtt_client_id", clientID, excludeID)
}

// ExistsMQTTUsername reports whether the MQTT username is taken.
func (s *Store) ExistsMQTTUsername(ctx context.Context, username, excludeID string) (bool, error) {
	return s.exists(ctx, "mqtt_username", username, excludeID)
}

// ExistingDeviceNames returns the lower-cased names already used in the model.
func (s *Store) ExistingDeviceNames(ctx context.Context, modelID string, names []string) (map[string]struct{}, error) {
	out := make(map[string]struct{}, len(names))
	if len(names) == 0 {
		return out, nil
	}
	args := make([]interface{}, 0, len(names)+1)
	args = append(args, modelID)
	for _, name := range names {
		args = append(args, name)
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT device_name FROM things WHERE thing_model_id = ? AND device_name IN (`+placeholders(len(names))+`)`,
		args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out[strings.ToLower(name)] = struct{}{}
	}
	return out, rows.Err()
}

// ExistingDeviceKeys returns the device keys that already exist.
func (s *Store) ExistingDeviceKeys(ctx context.Context, keys []string) (map[string]struct{}, error) {
	out := make(map[string]struct{}, len(keys))
	if len(keys) == 0 {
		return out, nil
	}
	args := make([]interface{}, 0, len(keys))
	for _, key := range keys {
		args = append(args, key)
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT device_key FROM things WHERE device_key IN (`+placeholders(len(keys))+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		out[key] = struct{}{}
	}
	return out, rows.Err()
}

func (s *Store) exists(ctx context.Context, column, value, excludeID string) (bool, error) {
	q, args := "SELECT COUNT(1) FROM things WHERE "+column+" = ?", []interface{}{value}
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

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanThing(scanner rowScanner) (*Thing, error) {
	item := &Thing{}
	var features []byte
	if err := scanner.Scan(
		&item.ID, &item.Version, &item.ThingModelID, &item.DeviceName, &item.DeviceKey,
		&item.DeviceSecret, &item.Token, &item.UpTopic, &item.DownTopic, &item.DeviceDesc,
		&features, &item.Image, &item.MQTTClientID, &item.MQTTUsername, &item.MQTTPassword, &item.Enable,
		&item.CreatedUser, &item.UpdatedUser, &item.CreatedTime, &item.UpdatedTime); err != nil {
		return nil, err
	}
	if len(features) > 0 {
		if err := json.Unmarshal(features, &item.Features); err != nil {
			return nil, err
		}
	}
	return item, nil
}

func marshalFeatureValues(features map[string]json.RawMessage) interface{} {
	if len(features) == 0 {
		return nil
	}
	raw, err := json.Marshal(features)
	if err != nil {
		return nil
	}
	return raw
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	var b strings.Builder
	b.Grow(n*2 - 1)
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('?')
	}
	return b.String()
}
