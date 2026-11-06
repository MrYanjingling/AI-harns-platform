package things

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"harns-platform/pkg/apis"
	"harns-platform/pkg/platform/thingmodel/tsl"
)

// Store is the SQL access layer of the things module.
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

const thingColumns = `id, version, thing_model_id, device_name, device_key, device_secret, token,
 up_topic, down_topic, device_desc, features, image, mqtt_client_id, mqtt_username, mqtt_password, enable, debug,
 created_user, updated_user, created_time, updated_time`

const authThingColumns = `t.id, t.version, t.thing_model_id, t.device_name, t.device_key, t.device_secret,
 t.token, t.up_topic, t.down_topic, t.device_desc, t.mqtt_client_id, t.mqtt_username,
 t.mqtt_password, t.enable, t.debug, t.created_user, t.updated_user, t.created_time, t.updated_time,
 m.thing_model_key`

const thingBasicColumns = `t.id, t.version, t.thing_model_id, t.device_name, t.device_key, t.device_secret,
 t.token, t.up_topic, t.down_topic, t.device_desc, t.features, t.image, t.mqtt_client_id,
 t.mqtt_username, t.mqtt_password, t.enable, t.debug, t.created_user, t.updated_user, t.created_time, t.updated_time,
 m.thing_model_key, m.name,
 cu.user_name, cu.nick_name, uu.user_name, uu.nick_name`

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

	var modelID string
	if err := tx.QueryRowContext(ctx,
		`SELECT id FROM thing_models WHERE id = ? FOR SHARE`, items[0].ThingModelID).Scan(&modelID); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO things (`+thingColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, item := range items {
		if _, err := stmt.ExecContext(ctx,
			item.ID, item.Version, item.ThingModelID, item.DeviceName, item.DeviceKey,
			item.DeviceSecret, item.Token, item.UpTopic, item.DownTopic, item.DeviceDesc,
			marshalFeatureValues(item.Features), item.Image,
			item.MQTTClientID, item.MQTTUsername, item.MQTTPassword, item.Enable, item.Debug,
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

// GetThingBasic loads a thing with model and creator/updater display fields.
func (s *Store) GetThingBasic(ctx context.Context, id string) (*ThingBasic, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+thingBasicColumns+`
		 FROM things t
		 JOIN thing_models m ON m.id = t.thing_model_id
		 LEFT JOIN users cu ON cu.id = t.created_user
		 LEFT JOIN users uu ON uu.id = t.updated_user
		 WHERE t.id = ?`, id)
	return scanThingBasic(row)
}

// GetThingSnapshotsByIDs loads credential-inclusive snapshots by thing ids.
func (s *Store) GetThingSnapshotsByIDs(ctx context.Context, ids []string) ([]*ThingSnapshot, error) {
	if len(ids) == 0 {
		return []*ThingSnapshot{}, nil
	}
	args := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT t.id, t.version, t.thing_model_id, t.device_name, t.device_key, t.device_secret, t.token,
		 t.up_topic, t.down_topic, t.device_desc, t.features, t.image, t.mqtt_client_id,
		 t.mqtt_username, t.mqtt_password, t.enable, t.debug,
		 t.created_user, t.updated_user, t.created_time, t.updated_time, m.thing_model_key
		 FROM things t JOIN thing_models m ON m.id = t.thing_model_id
		 WHERE t.id IN (`+placeholders(len(ids))+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	snapshots := make([]*ThingSnapshot, 0, len(ids))
	for rows.Next() {
		snapshot, err := scanThingSnapshot(rows)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, rows.Err()
}

// GetThingIDByDeviceKey resolves a device key to its thing id.
func (s *Store) GetThingIDByDeviceKey(ctx context.Context, deviceKey string) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM things WHERE device_key = ?`, deviceKey).Scan(&id)
	return id, err
}

// AuthenticateThing loads a thing matching the MQTT credentials.
func (s *Store) AuthenticateThing(ctx context.Context, clientID, username, password string) (*AuthThing, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+authThingColumns+`
		 FROM things t JOIN thing_models m ON m.id = t.thing_model_id
		 WHERE t.mqtt_client_id = ? AND t.mqtt_username = ? AND t.mqtt_password = ?
		 LIMIT 1`, clientID, username, password)

	item := &AuthThing{}
	if err := row.Scan(
		&item.ID, &item.Version, &item.ThingModelID, &item.DeviceName, &item.DeviceKey,
		&item.DeviceSecret, &item.Token, &item.UpTopic, &item.DownTopic, &item.DeviceDesc,
		&item.MQTTClientID, &item.MQTTUsername, &item.MQTTPassword, &item.Enable, &item.Debug,
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

// GetThingModelPropertySamples returns the first properties for upload samples.
func (s *Store) GetThingModelPropertySamples(ctx context.Context, modelID string, limit int) ([]tsl.Property, error) {
	if limit <= 0 {
		limit = 3
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT identifier, data_type FROM thing_model_properties
		 WHERE thing_model_id = ?
		 ORDER BY sort ASC, created_time ASC, id ASC
		 LIMIT ?`, modelID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	properties := make([]tsl.Property, 0, limit)
	for rows.Next() {
		var (
			identifier string
			dataType   []byte
		)
		if err := rows.Scan(&identifier, &dataType); err != nil {
			return nil, err
		}
		property := tsl.Property{Identifier: identifier}
		if len(dataType) > 0 {
			if err := json.Unmarshal(dataType, &property.DataType); err != nil {
				return nil, err
			}
		}
		properties = append(properties, property)
	}
	return properties, rows.Err()
}

// ListThingModelProperties returns a filtered page of property definitions for a model.
func (s *Store) ListThingModelProperties(ctx context.Context, modelID string, q apis.PageQuery, keyword string) ([]tsl.Property, int64, error) {
	where := "thing_model_id = ?"
	args := []interface{}{modelID}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		where += " AND (identifier LIKE ? OR name LIKE ?)"
		kw := "%" + keyword + "%"
		args = append(args, kw, kw)
	}

	var total int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM thing_model_properties WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT identifier, name, data_type, rw_flag, required, description, custom, sort
		 FROM thing_model_properties
		 WHERE `+where+`
		 ORDER BY sort ASC, created_time ASC, id ASC
		 LIMIT ? OFFSET ?`, append(args, q.Limit(), q.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	properties := make([]tsl.Property, 0, q.Limit())
	for rows.Next() {
		var (
			property tsl.Property
			dataType []byte
			required int
			custom   []byte
		)
		if err := rows.Scan(
			&property.Identifier, &property.Name, &dataType, &property.RWFlag,
			&required, &property.Description, &custom, &property.Sort); err != nil {
			return nil, 0, err
		}
		property.Required = required != 0
		if len(dataType) > 0 {
			if err := json.Unmarshal(dataType, &property.DataType); err != nil {
				return nil, 0, err
			}
		}
		if len(custom) > 0 {
			if err := json.Unmarshal(custom, &property.Custom); err != nil {
				return nil, 0, err
			}
		}
		properties = append(properties, property)
	}
	return properties, total, rows.Err()
}

// GetThingByToken loads an enabled thing identity by HTTP token.
func (s *Store) GetThingByToken(ctx context.Context, token string) (*ThingIdentity, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT t.id, t.thing_model_id, m.thing_model_key, t.device_key, t.device_name,
		 t.mqtt_client_id, t.mqtt_username, t.up_topic, t.down_topic, t.enable, t.debug
		 FROM things t JOIN thing_models m ON m.id = t.thing_model_id
		 WHERE t.token = ? AND t.enable = 1 LIMIT 1`, token)
	item := &ThingIdentity{}
	if err := row.Scan(
		&item.ID, &item.ThingModelID, &item.ThingModelKey, &item.DeviceKey, &item.DeviceName,
		&item.MQTTClientID, &item.MQTTUsername, &item.UpTopic, &item.DownTopic,
		&item.Enable, &item.Debug); err != nil {
		return nil, err
	}
	return item, nil
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
		 mqtt_password=?, enable=?, debug=?, updated_user=?, updated_time=?, version=version+1 WHERE id=?`,
		item.DeviceName, item.DeviceDesc, marshalFeatureValues(item.Features), item.Image, item.MQTTClientID, item.MQTTUsername,
		item.MQTTPassword, item.Enable, item.Debug, item.UpdatedUser, item.UpdatedTime, item.ID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// DeleteThings deletes things by id and returns the removed rows needed for
// cache cleanup. Missing ids are ignored.
func (s *Store) DeleteThings(ctx context.Context, ids []string) (int64, []DeletedThingRef, error) {
	if len(ids) == 0 {
		return 0, nil, nil
	}
	args := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx,
		`SELECT id, token, mqtt_client_id FROM things WHERE id IN (`+placeholders(len(ids))+`) FOR UPDATE`, args...)
	if err != nil {
		return 0, nil, err
	}
	refs := make([]DeletedThingRef, 0, len(ids))
	for rows.Next() {
		var ref DeletedThingRef
		if err := rows.Scan(&ref.ID, &ref.Token, &ref.MQTTClientID); err != nil {
			rows.Close()
			return 0, nil, err
		}
		refs = append(refs, ref)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, nil, err
	}
	if err := rows.Close(); err != nil {
		return 0, nil, err
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM things WHERE id IN (`+placeholders(len(ids))+`)`, args...)
	if err != nil {
		return 0, nil, err
	}
	deleted, err := res.RowsAffected()
	if err != nil {
		return 0, nil, err
	}
	if err := tx.Commit(); err != nil {
		return 0, nil, err
	}
	return deleted, refs, nil
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

func scanThingSnapshot(scanner rowScanner) (*ThingSnapshot, error) {
	item := &ThingSnapshot{}
	var features []byte
	if err := scanner.Scan(
		&item.ID, &item.Version, &item.ThingModelID, &item.DeviceName, &item.DeviceKey,
		&item.DeviceSecret, &item.Token, &item.UpTopic, &item.DownTopic, &item.DeviceDesc,
		&features, &item.Image, &item.MQTTClientID, &item.MQTTUsername, &item.MQTTPassword,
		&item.Enable, &item.Debug, &item.CreatedUser, &item.UpdatedUser, &item.CreatedTime,
		&item.UpdatedTime, &item.ThingModelKey); err != nil {
		return nil, err
	}
	if len(features) > 0 {
		if err := json.Unmarshal(features, &item.Features); err != nil {
			return nil, err
		}
	}
	return item, nil
}

func scanThing(scanner rowScanner) (*Thing, error) {
	item := &Thing{}
	var features []byte
	if err := scanner.Scan(
		&item.ID, &item.Version, &item.ThingModelID, &item.DeviceName, &item.DeviceKey,
		&item.DeviceSecret, &item.Token, &item.UpTopic, &item.DownTopic, &item.DeviceDesc,
		&features, &item.Image, &item.MQTTClientID, &item.MQTTUsername, &item.MQTTPassword, &item.Enable, &item.Debug,
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

func scanThingBasic(scanner rowScanner) (*ThingBasic, error) {
	item := &ThingBasic{}
	var (
		features        []byte
		createdUserName sql.NullString
		createdNickName sql.NullString
		updatedUserName sql.NullString
		updatedNickName sql.NullString
	)
	if err := scanner.Scan(
		&item.ID, &item.Version, &item.ThingModelID, &item.DeviceName, &item.DeviceKey,
		&item.DeviceSecret, &item.Token, &item.UpTopic, &item.DownTopic, &item.DeviceDesc,
		&features, &item.Image, &item.MQTTClientID, &item.MQTTUsername, &item.MQTTPassword, &item.Enable, &item.Debug,
		&item.CreatedUser, &item.UpdatedUser, &item.CreatedTime, &item.UpdatedTime,
		&item.ThingModelKey, &item.ThingModelName,
		&createdUserName, &createdNickName, &updatedUserName, &updatedNickName); err != nil {
		return nil, err
	}
	if len(features) > 0 {
		if err := json.Unmarshal(features, &item.Features); err != nil {
			return nil, err
		}
	}
	item.CreatedUserName = createdUserName.String
	item.CreatedUserNickName = createdNickName.String
	item.UpdatedUserName = updatedUserName.String
	item.UpdatedUserNickName = updatedNickName.String
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
