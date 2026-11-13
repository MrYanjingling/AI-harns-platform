package usermgr

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"harns-platform/pkg/apis"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    username      TEXT NOT NULL COLLATE NOCASE UNIQUE,
    password_hash TEXT NOT NULL,
    remark        TEXT NOT NULL DEFAULT '',
    is_admin      INTEGER NOT NULL DEFAULT 0,
    version       INTEGER NOT NULL DEFAULT 1,
    created_time  TEXT NOT NULL,
    updated_time  TEXT NOT NULL,
    CHECK (is_admin IN (0, 1)),
    CHECK (username <> '')
);
CREATE INDEX IF NOT EXISTS idx_users_created_time ON users (created_time);
`

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Initialize(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("user store database is nil")
	}
	_, err := s.db.ExecContext(ctx, schemaSQL)
	return err
}

func (s *Store) List(ctx context.Context, q apis.PageQuery, keyword string) ([]*User, int64, error) {
	where := "1=1"
	args := make([]interface{}, 0, 2)
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		where = "(username LIKE ? OR remark LIKE ?)"
		like := "%" + keyword + "%"
		args = append(args, like, like)
	}
	var total int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM users WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	queryArgs := append(append([]interface{}{}, args...), q.Limit(), q.Offset())
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, username, password_hash, remark, is_admin, version, created_time, updated_time
		 FROM users WHERE `+where+` ORDER BY created_time ASC, id ASC LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]*User, 0)
	for rows.Next() {
		item, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, rows.Err()
}

func (s *Store) Get(ctx context.Context, id string) (*User, error) {
	return scanUser(s.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, remark, is_admin, version, created_time, updated_time
		 FROM users WHERE id = ?`, id))
}

func (s *Store) GetByUsername(ctx context.Context, username string) (*User, error) {
	return scanUser(s.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, remark, is_admin, version, created_time, updated_time
		 FROM users WHERE username = ?`, username))
}

func (s *Store) UsernameExists(ctx context.Context, username, excludeID string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM users WHERE username = ? AND id <> ?`, username, excludeID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) HasSuperuser(ctx context.Context) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM users WHERE is_admin = 1`).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) Create(ctx context.Context, item *User) error {
	if item == nil {
		return fmt.Errorf("user is nil")
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, username, password_hash, remark, is_admin, version, created_time, updated_time)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.Username, item.PasswordHash, item.Remark, boolInt(item.IsAdmin), item.Version,
		item.CreatedTime.UTC().Format(time.RFC3339Nano), item.UpdatedTime.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) Update(ctx context.Context, item *User) error {
	if item == nil {
		return fmt.Errorf("user is nil")
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE users SET username = ?, remark = ?, version = version + 1, updated_time = ? WHERE id = ?`,
		item.Username, item.Remark, item.UpdatedTime.UTC().Format(time.RFC3339Nano), item.ID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (s *Store) UpdatePassword(ctx context.Context, id, hash string, now time.Time) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE users SET password_hash = ?, version = version + 1, updated_time = ? WHERE id = ?`,
		hash, now.UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanUser(row rowScanner) (*User, error) {
	var (
		item       User
		isAdmin    int
		createdRaw string
		updatedRaw string
	)
	if err := row.Scan(&item.ID, &item.Username, &item.PasswordHash, &item.Remark, &isAdmin,
		&item.Version, &createdRaw, &updatedRaw); err != nil {
		return nil, err
	}
	created, err := time.Parse(time.RFC3339Nano, createdRaw)
	if err != nil {
		return nil, err
	}
	updated, err := time.Parse(time.RFC3339Nano, updatedRaw)
	if err != nil {
		return nil, err
	}
	item.IsAdmin = isAdmin != 0
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
