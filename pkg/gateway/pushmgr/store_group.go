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

func (s *Store) ListAllGroups(ctx context.Context) ([]*PushGroup, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, connector_id, data_topic, command_topic, run_state, push_interval_ms, all_collections, max_collections_per_push_message, version, created_time, updated_time
		 FROM push_groups ORDER BY created_time ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	out := make([]*PushGroup, 0)
	for rows.Next() {
		item, err := scanGroup(rows)
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for _, item := range out {
		if err := s.loadGroupCollectionIDs(ctx, item); err != nil {
			return nil, err
		}
	}
	return out, nil
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
		`SELECT id, name, connector_id, data_topic, command_topic, run_state, push_interval_ms, all_collections, max_collections_per_push_message, version, created_time, updated_time
		 FROM push_groups WHERE `+where+` ORDER BY created_time ASC, id ASC LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	out := make([]*PushGroup, 0)
	for rows.Next() {
		item, err := scanGroup(rows)
		if err != nil {
			_ = rows.Close()
			return nil, 0, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, 0, err
	}
	if err := rows.Close(); err != nil {
		return nil, 0, err
	}
	for _, item := range out {
		if err := s.loadGroupCollectionIDs(ctx, item); err != nil {
			return nil, 0, err
		}
	}
	return out, total, nil
}

func (s *Store) GetGroup(ctx context.Context, id string) (*PushGroup, error) {
	item, err := scanGroup(s.db.QueryRowContext(ctx,
		`SELECT id, name, connector_id, data_topic, command_topic, run_state, push_interval_ms, all_collections, max_collections_per_push_message, version, created_time, updated_time
		 FROM push_groups WHERE id = ?`, id))
	if err != nil {
		return nil, err
	}
	if err := s.loadGroupCollectionIDs(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Store) loadGroupCollectionIDs(ctx context.Context, item *PushGroup) error {
	if item == nil {
		return nil
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT collection_id FROM push_group_collections WHERE push_group_id = ? ORDER BY rowid ASC`,
		item.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	item.CollectionIDs = make([]string, 0)
	for rows.Next() {
		var collectionID string
		if err := rows.Scan(&collectionID); err != nil {
			return err
		}
		item.CollectionIDs = append(item.CollectionIDs, collectionID)
	}
	return rows.Err()
}

func (s *Store) CreateGroup(ctx context.Context, item *PushGroup) error {
	if item == nil {
		return fmt.Errorf("push group is nil")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO push_groups (id, name, connector_id, data_topic, command_topic, run_state, push_interval_ms, all_collections, max_collections_per_push_message, version, created_time, updated_time)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.Name, item.ConnectorID, item.DataTopic, item.CommandTopic, item.RunState, item.PushIntervalMs, boolInt(item.AllCollections),
		item.MaxCollectionsPerPushMessage, item.Version, item.CreatedTime.UTC().Format(time.RFC3339Nano), item.UpdatedTime.UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	if err := replaceGroupCollectionLinks(ctx, tx, item.ID, item.CollectionIDs); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) UpdateGroup(ctx context.Context, item *PushGroup) error {
	if item == nil {
		return fmt.Errorf("push group is nil")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx,
		`UPDATE push_groups SET name = ?, connector_id = ?, data_topic = ?, command_topic = ?, run_state = ?, push_interval_ms = ?, all_collections = ?, max_collections_per_push_message = ?, version = ?, updated_time = ? WHERE id = ?`,
		item.Name, item.ConnectorID, item.DataTopic, item.CommandTopic, item.RunState, item.PushIntervalMs, boolInt(item.AllCollections),
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
	if err := replaceGroupCollectionLinks(ctx, tx, item.ID, item.CollectionIDs); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) DeleteGroup(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM ready_send WHERE group_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM push_group_collections WHERE push_group_id = ?`, id); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM push_groups WHERE id = ?`, id)
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
	return tx.Commit()
}

func replaceGroupCollectionLinks(ctx context.Context, tx *sql.Tx, groupID string, collectionIDs []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM push_group_collections WHERE push_group_id = ?`, groupID); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(collectionIDs))
	for _, collectionID := range collectionIDs {
		collectionID = strings.TrimSpace(collectionID)
		if collectionID == "" {
			continue
		}
		if _, ok := seen[collectionID]; ok {
			continue
		}
		seen[collectionID] = struct{}{}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO push_group_collections (push_group_id, collection_id) VALUES (?, ?)`,
			groupID, collectionID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) RemoveCollectionReferences(ctx context.Context, collectionID string) ([]string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx,
		`SELECT DISTINCT push_group_id FROM push_group_collections WHERE collection_id = ?`,
		collectionID)
	if err != nil {
		return nil, err
	}
	groupIDs := make([]string, 0)
	for rows.Next() {
		var groupID string
		if err := rows.Scan(&groupID); err != nil {
			_ = rows.Close()
			return nil, err
		}
		groupIDs = append(groupIDs, groupID)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM push_group_collections WHERE collection_id = ?`, collectionID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return groupIDs, nil
}

func scanGroup(row rowScanner) (*PushGroup, error) {
	var (
		item       PushGroup
		all        int
		createdRaw string
		updatedRaw string
	)
	if err := row.Scan(&item.ID, &item.Name, &item.ConnectorID, &item.DataTopic, &item.CommandTopic, &item.RunState, &item.PushIntervalMs, &all, &item.MaxCollectionsPerPushMessage, &item.Version, &createdRaw, &updatedRaw); err != nil {
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
