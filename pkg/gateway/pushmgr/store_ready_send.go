package pushmgr

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func (s *Store) SaveReadySend(ctx context.Context, batch *ReadySendBatch) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("push store database is nil")
	}
	if batch == nil || batch.GroupID == "" {
		return fmt.Errorf("ready_send batch is nil")
	}
	if len(batch.Payload) == 0 {
		return fmt.Errorf("ready_send payload is empty")
	}
	if batch.PayloadSize <= 0 {
		return fmt.Errorf("ready_send payload size is invalid")
	}
	if batch.CreatedTime.IsZero() {
		batch.CreatedTime = time.Now().UTC()
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx,
		`INSERT INTO ready_send (group_id, connector_id, topic, payload, payload_size, created_time)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		batch.GroupID, batch.ConnectorID, batch.Topic, batch.Payload, batch.PayloadSize,
		batch.CreatedTime.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	if batch.Seq == 0 {
		batch.Seq, _ = res.LastInsertId()
	}
	if err := trimReadySendTx(ctx, tx, batch.GroupID); err != nil {
		return err
	}
	return tx.Commit()
}

// ListReadySend returns pending rows in insertion order.
func (s *Store) ListReadySend(ctx context.Context, groupID string, limit int) ([]*ReadySendBatch, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("push store database is nil")
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT seq, group_id, connector_id, topic, payload, payload_size, created_time
		 FROM ready_send WHERE group_id = ? ORDER BY seq ASC LIMIT ?`, groupID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*ReadySendBatch, 0, limit)
	for rows.Next() {
		item, err := scanReadySend(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// DeleteReadySend removes one pending row after a successful or discarded send.
func (s *Store) DeleteReadySend(ctx context.Context, groupID string, seq int64) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("push store database is nil")
	}
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM ready_send WHERE group_id = ? AND seq = ?`, groupID, seq)
	return err
}

// ReadySendStats returns persisted queue statistics for runtime counter seeding.
func (s *Store) ReadySendStats(ctx context.Context, groupID string) (ReadySendStats, error) {
	if s == nil || s.db == nil {
		return ReadySendStats{}, fmt.Errorf("push store database is nil")
	}
	var (
		messages int64
		batches  int64
		oldest   sql.NullString
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(payload_size), 0), COUNT(1), MIN(created_time)
		 FROM ready_send WHERE group_id = ?`, groupID).Scan(&messages, &batches, &oldest)
	if err != nil {
		return ReadySendStats{}, err
	}
	stats := ReadySendStats{Messages: messages, Batches: batches}
	if oldest.Valid && oldest.String != "" {
		if value, parseErr := time.Parse(time.RFC3339Nano, oldest.String); parseErr == nil {
			stats.Oldest = value
		}
	}
	return stats, nil
}

func scanReadySend(row rowScanner) (*ReadySendBatch, error) {
	var (
		item       ReadySendBatch
		createdRaw string
	)
	if err := row.Scan(&item.Seq, &item.GroupID, &item.ConnectorID, &item.Topic,
		&item.Payload, &item.PayloadSize, &createdRaw); err != nil {
		return nil, err
	}
	if value, err := time.Parse(time.RFC3339Nano, createdRaw); err == nil {
		item.CreatedTime = value
	}
	return &item, nil
}

func trimReadySendTx(ctx context.Context, tx *sql.Tx, groupID string) error {
	rows, err := tx.QueryContext(ctx,
		`SELECT seq, payload_size, LENGTH(payload), created_time
		 FROM ready_send WHERE group_id = ? ORDER BY seq ASC`, groupID)
	if err != nil {
		return err
	}
	type row struct {
		seq     int64
		count   int64
		bytes   int64
		created time.Time
	}
	items := make([]row, 0)
	for rows.Next() {
		var (
			item       row
			createdRaw string
		)
		if err := rows.Scan(&item.seq, &item.count, &item.bytes, &createdRaw); err != nil {
			_ = rows.Close()
			return err
		}
		if value, parseErr := time.Parse(time.RFC3339Nano, createdRaw); parseErr == nil {
			item.created = value
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	var totalMessages, totalBytes int64
	for _, item := range items {
		totalMessages += item.count
		totalBytes += item.bytes
	}
	cutoff := time.Now().UTC().Add(-DefaultReadySendMaxAge)
	for _, item := range items {
		expired := !item.created.IsZero() && item.created.Before(cutoff)
		if !expired && totalMessages <= DefaultReadySendMaxMessages && totalBytes <= DefaultReadySendMaxBytes {
			break
		}
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM ready_send WHERE group_id = ? AND seq = ?`, groupID, item.seq); err != nil {
			return err
		}
		totalMessages -= item.count
		totalBytes -= item.bytes
	}
	return nil
}
