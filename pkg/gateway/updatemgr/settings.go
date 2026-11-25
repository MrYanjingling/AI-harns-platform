package updatemgr

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	cloudSettingsTableSQL = `
CREATE TABLE IF NOT EXISTS cloud_settings (
    singleton       INTEGER PRIMARY KEY CHECK (singleton = 1),
    enabled         INTEGER NOT NULL DEFAULT 0,
    broker_url      TEXT NOT NULL DEFAULT '',
    username        TEXT NOT NULL DEFAULT '',
    password_cipher BLOB,
    updated_time    TEXT NOT NULL,
    CHECK (enabled IN (0, 1))
);
`
	settingsPayloadVersion = byte(1)
	settingsPayloadAAD     = "harns-gateway-cloud-v1"
)

// Settings is the persisted cloud management and MQTT configuration.
type Settings struct {
	Enabled     bool
	BrokerURL   string
	Username    string
	Password    string
	UpdatedTime time.Time
}

// SettingsVo is the safe settings response. Passwords are never returned.
type SettingsVo struct {
	Enabled     bool      `json:"enabled"`
	BrokerURL   string    `json:"brokerUrl"`
	Username    string    `json:"username"`
	HasPassword bool      `json:"hasPassword"`
	UpdatedTime time.Time `json:"updatedTime"`
}

// UpdateSettingsReq updates cloud management and MQTT settings.
type UpdateSettingsReq struct {
	Enabled   bool   `json:"enabled"`
	BrokerURL string `json:"brokerUrl"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

// View is the API response for the cloud-managed OTA page.
type View struct {
	Installed Installed  `json:"installed"`
	Task      *Task      `json:"task,omitempty"`
	Settings  SettingsVo `json:"settings"`
}

type settingsStore struct {
	db    *sql.DB
	codec *settingsCodec
}

func newSettingsStore(db *sql.DB, secret string) (*settingsStore, error) {
	codec, err := newSettingsCodec(secret)
	if err != nil {
		return nil, err
	}
	return &settingsStore{db: db, codec: codec}, nil
}

func (s *settingsStore) Initialize(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("cloud settings store database is nil")
	}
	if _, err := s.db.ExecContext(ctx, cloudSettingsTableSQL); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO cloud_settings (singleton, enabled, broker_url, username, password_cipher, updated_time)
		 VALUES (1, 0, '', '', NULL, ?)`, formatSettingsTime(time.Now().UTC()))
	return err
}

func (s *settingsStore) Get(ctx context.Context) (*Settings, error) {
	var (
		item        Settings
		enabled     int
		cipherBytes []byte
		updatedTime string
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT enabled, broker_url, username, password_cipher, updated_time
		 FROM cloud_settings WHERE singleton = 1`).Scan(
		&enabled, &item.BrokerURL, &item.Username, &cipherBytes, &updatedTime)
	if err != nil {
		return nil, err
	}
	item.Enabled = enabled == 1
	item.Password, err = s.codec.decryptString(cipherBytes)
	if err != nil {
		return nil, err
	}
	item.UpdatedTime, err = time.Parse(time.RFC3339Nano, updatedTime)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *settingsStore) Update(ctx context.Context, item *Settings) error {
	if item == nil {
		return fmt.Errorf("cloud settings is nil")
	}
	now := time.Now().UTC()
	item.UpdatedTime = now
	cipherBytes, err := s.codec.encryptString(item.Password)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx,
		`UPDATE cloud_settings
		 SET enabled = ?, broker_url = ?, username = ?, password_cipher = ?, updated_time = ?
		 WHERE singleton = 1`,
		boolInt(item.Enabled), strings.TrimSpace(item.BrokerURL), strings.TrimSpace(item.Username),
		cipherBytes, formatSettingsTime(now))
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s Settings) toVo() SettingsVo {
	return SettingsVo{
		Enabled:     s.Enabled,
		BrokerURL:   s.BrokerURL,
		Username:    s.Username,
		HasPassword: s.Password != "",
		UpdatedTime: s.UpdatedTime,
	}
}

func formatSettingsTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

type settingsCodec struct {
	aead cipher.AEAD
}

func newSettingsCodec(secret string) (*settingsCodec, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("OTA encryption secret is empty")
	}
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("create OTA cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create OTA GCM: %w", err)
	}
	return &settingsCodec{aead: aead}, nil
}

func (c *settingsCodec) encryptString(value string) ([]byte, error) {
	if value == "" {
		return nil, nil
	}
	nonceSize := c.aead.NonceSize()
	data := make([]byte, 1+nonceSize, 1+nonceSize+len(value)+c.aead.Overhead())
	data[0] = settingsPayloadVersion
	if _, err := rand.Read(data[1 : 1+nonceSize]); err != nil {
		return nil, fmt.Errorf("generate OTA nonce: %w", err)
	}
	return c.aead.Seal(data, data[1:1+nonceSize], []byte(value), []byte(settingsPayloadAAD)), nil
}

func (c *settingsCodec) decryptString(data []byte) (string, error) {
	if len(data) == 0 {
		return "", nil
	}
	nonceSize := c.aead.NonceSize()
	if len(data) < 1+nonceSize+c.aead.Overhead() {
		return "", fmt.Errorf("OTA payload is too short")
	}
	if data[0] != settingsPayloadVersion {
		return "", fmt.Errorf("unsupported OTA payload version %d", data[0])
	}
	plain, err := c.aead.Open(nil, data[1:1+nonceSize], data[1+nonceSize:], []byte(settingsPayloadAAD))
	if err != nil {
		return "", fmt.Errorf("decrypt OTA payload: %w", err)
	}
	return string(plain), nil
}
