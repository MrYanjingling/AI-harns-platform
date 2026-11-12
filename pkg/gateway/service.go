package gateway

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"harns-platform/pkg/gateway/runtime"
	"harns-platform/pkg/utils/randutil"
	"harns-platform/pkg/utils/uuidutil"
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
	meta, err := s.store.GetMeta(ctx)
	if err == nil {
		return meta, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	meta = &GatewayMeta{
		Secret: "",
		ObjectMeta: runtime.ObjectMeta{
			Name:    "harnsgateway",
			ID:      uuidutil.UUID(),
			Version: strconv.FormatUint(randutil.Uint64n(), 10),
			ModTime: time.Now().UTC(),
		},
	}
	if err := s.store.CreateMeta(ctx, meta); err != nil {
		return nil, err
	}
	return meta, nil
}
