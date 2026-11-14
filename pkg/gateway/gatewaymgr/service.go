package gatewaymgr

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"harns-platform/pkg/gateway/runtime"
	"harns-platform/pkg/gateway/validation"
	"harns-platform/pkg/utils/uuidutil"
)

var gatewayNamePattern = regexp.MustCompile(`^[\p{L}_0-9_. ]+$`)

// Service manages gateway metadata.
type Service struct {
	store *Store
}

// NewService creates a gateway metadata service.
func NewService(store *Store) *Service {
	return &Service{store: store}
}

// LoadOrCreate loads gateway metadata, creating it when the database is empty.
func (s *Service) LoadOrCreate(ctx context.Context) (*GatewayMeta, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("gateway service store is nil")
	}
	now := time.Now().UTC()
	meta, err := s.store.GetMeta(ctx)
	if err == nil {
		if meta.Secret == "" {
			secret, err := generateGatewaySecret()
			if err != nil {
				return nil, err
			}
			meta.Secret = secret
		}
		if meta.Version == "" || isLegacyCommitETag(meta.Version) {
			meta.Version = uuidutil.UUID()
		}
		meta.ModTime = now
		if err := s.store.UpdateMeta(ctx, meta); err != nil {
			return nil, err
		}
		return meta, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	secret, err := generateGatewaySecret()
	if err != nil {
		return nil, err
	}
	meta = &GatewayMeta{
		Secret:   secret,
		InitTime: now,
		ObjectMeta: runtime.ObjectMeta{
			Name:    "harnsgateway",
			ID:      uuidutil.UUID(),
			Version: uuidutil.UUID(),
			ModTime: now,
		},
	}
	if err := s.store.CreateMeta(ctx, meta); err != nil {
		return nil, err
	}
	return meta, nil
}

func (s *Service) Rename(ctx context.Context, id, name string) (string, error) {
	if s == nil || s.store == nil {
		return "", fmt.Errorf("gateway service store is nil")
	}
	name = strings.TrimSpace(name)
	if err := validateGatewayName(name); err != nil {
		return "", err
	}
	eTag := uuidutil.UUID()
	if err := s.store.UpdateName(ctx, id, name, eTag); err != nil {
		return "", err
	}
	return eTag, nil
}

func isLegacyCommitETag(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, char := range value {
		if !strings.ContainsRune("0123456789abcdefABCDEF", char) {
			return false
		}
	}
	return true
}

func validateGatewayName(name string) error {
	issues := make([]validation.FieldIssue, 0, 1)
	switch {
	case name == "":
		issues = append(issues, validation.FieldIssue{Field: "name", MsgID: "validate.gateway_name_required"})
	case utf8.RuneCountInString(name) > 64 || !gatewayNamePattern.MatchString(name):
		issues = append(issues, validation.FieldIssue{Field: "name", MsgID: "validate.gateway_name_invalid"})
	}
	if err := validation.New(issues); err != nil {
		return err
	}
	return nil
}

func generateGatewaySecret() (string, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("generate gateway secret: %w", err)
	}
	return hex.EncodeToString(secret), nil
}
