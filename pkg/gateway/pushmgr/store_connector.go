package pushmgr

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"harns-platform/pkg/apis"
)

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
	if _, err := tx.ExecContext(ctx, `DELETE FROM ready_send WHERE connector_id = ?`, id); err != nil {
		return nil, err
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
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM push_group_collections WHERE push_group_id IN (SELECT id FROM push_groups WHERE connector_id = ?)`,
			id); err != nil {
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
