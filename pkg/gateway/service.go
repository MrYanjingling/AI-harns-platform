package gateway

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"harns-platform/pkg/gateway/runtime"
	"harns-platform/pkg/utils/uuidutil"
	"harns-platform/pkg/version"
)

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
		meta.ModTime = now
		meta.Version = version.CommitID()
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
			Version: version.CommitID(),
			ModTime: now,
		},
	}
	if err := s.store.CreateMeta(ctx, meta); err != nil {
		return nil, err
	}
	return meta, nil
}

func generateGatewaySecret() (string, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("generate gateway secret: %w", err)
	}
	return hex.EncodeToString(secret), nil
}
