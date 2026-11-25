package licensemgr

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"harns-platform/pkg/gateway/validation"
	"harns-platform/pkg/logging"
)

// IdentityProvider returns the current gateway id and secret.
type IdentityProvider func() (gatewayID, secret string, err error)

// InitTimeProvider returns the gateway initialization time.
type InitTimeProvider func(context.Context) (time.Time, error)

// Service validates licenses and keeps the active license in memory.
type Service struct {
	store         *Store
	identity      IdentityProvider
	initTime      InitTimeProvider
	publicKeyFile string

	mu        sync.RWMutex
	claims    *Claims
	trialInfo *Info
}

// NewService creates a license service.
func NewService(store *Store, identity IdentityProvider, publicKeyFile string) *Service {
	return &Service{
		store:         store,
		identity:      identity,
		publicKeyFile: strings.TrimSpace(publicKeyFile),
	}
}

// SetInitTimeProvider sets the gateway initialization time provider.
func (s *Service) SetInitTimeProvider(provider InitTimeProvider) {
	if s != nil {
		s.initTime = provider
	}
}

// Load restores the persisted license into memory.
func (s *Service) Load(ctx context.Context) error {
	defer logging.Trace("licensemgr.Load")()
	if s == nil || s.store == nil {
		return fmt.Errorf("license service store is nil")
	}
	s.loadTrialInfo(ctx)
	data, err := s.store.Get(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	claims, err := s.parseAndMatch(data)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.claims = claims
	s.mu.Unlock()
	return nil
}

// Import verifies and persists a license, then updates the memory cache.
func (s *Service) Import(ctx context.Context, data string) (*Status, error) {
	defer logging.Trace("licensemgr.Import", "dataLength", len(data))()
	data = strings.TrimSpace(data)
	if data == "" {
		return nil, validation.New([]validation.FieldIssue{{
			Field: "data", MsgID: "validate.license_data_required",
		}})
	}
	claims, err := s.parseAndMatch(data)
	if err != nil {
		return nil, err
	}
	if err := ValidateForImport(claims, time.Now().UTC()); err != nil {
		return nil, err
	}
	if err := s.store.Save(ctx, data); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.claims = claims
	s.mu.Unlock()
	return &Status{
		Status:  StatusAuthorized,
		License: claimsToInfo(claims),
	}, nil
}

// GetStatus returns the current authorization status from memory.
func (s *Service) GetStatus() *Status {
	defer logging.Trace("licensemgr.GetStatus")()
	if s == nil {
		return &Status{Status: StatusUnauthorized}
	}
	s.mu.RLock()
	claims := s.claims
	s.mu.RUnlock()
	if claims == nil {
		return &Status{Status: StatusUnauthorized, License: s.ensureTrialInfo()}
	}
	if ValidateForImport(claims, time.Now().UTC()) != nil {
		return &Status{Status: StatusExpired, License: claimsToInfo(claims)}
	}
	return &Status{
		Status:  StatusAuthorized,
		License: claimsToInfo(claims),
	}
}

// IsAuthorized reports whether the current license status is authorized.
func (s *Service) IsAuthorized() bool {
	if s == nil {
		return false
	}
	return s.GetStatus().Status == StatusAuthorized
}

// MaxVariable returns the current licensed collection-variable quota.
func (s *Service) MaxVariable() int {
	s.mu.RLock()
	claims := s.claims
	s.mu.RUnlock()
	if claims != nil && ValidateForImport(claims, time.Now().UTC()) == nil {
		return claims.MaxVariable
	}
	info := s.ensureTrialInfo()
	if info == nil {
		return 0
	}
	return info.MaxVariable
}

// SoftwareRunDeadline returns the current license software run deadline.
// The persisted claims are used even after license expiry so the guard can
// still honor the signed run deadline. Without claims, the trial deadline is used.
func (s *Service) SoftwareRunDeadline() (time.Time, bool) {
	if s == nil {
		return time.Time{}, false
	}
	s.mu.RLock()
	claims := s.claims
	s.mu.RUnlock()
	if claims != nil && !claims.SoftwareRunDeadline.IsZero() {
		return claims.SoftwareRunDeadline.UTC(), true
	}
	info := s.ensureTrialInfo()
	if info == nil || strings.TrimSpace(info.SoftwareRunDeadline) == "" {
		return time.Time{}, false
	}
	deadline, err := time.Parse(time.RFC3339, info.SoftwareRunDeadline)
	if err != nil {
		return time.Time{}, false
	}
	return deadline.UTC(), true
}

// ensureTrialInfo returns the trial quota and deadline, loading it on demand so
// unauthorized responses always carry the default quota and run deadline.
func (s *Service) ensureTrialInfo() *Info {
	s.mu.RLock()
	info := s.trialInfo
	s.mu.RUnlock()
	if info != nil {
		return info
	}
	s.loadTrialInfo(context.Background())
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.trialInfo
}

func (s *Service) loadTrialInfo(ctx context.Context) {
	initTime := time.Now().UTC()
	if s.initTime != nil {
		value, err := s.initTime(ctx)
		if err != nil {
			logging.Warnf("Failed to load gateway init time for trial license: %v", err)
		} else {
			initTime = value
		}
	}
	info := &Info{
		MaxVariable:         DefaultTrialMaxVariable,
		SoftwareRunDeadline: initTime.Add(DefaultTrialDuration).UTC().Format(time.RFC3339),
	}
	s.mu.Lock()
	s.trialInfo = info
	s.mu.Unlock()
}

func (s *Service) parseAndMatch(data string) (*Claims, error) {
	if s == nil || s.identity == nil {
		return nil, fmt.Errorf("license service identity provider is nil")
	}
	publicKey, err := LoadPublicKey(s.publicKeyFile)
	if err != nil {
		return nil, err
	}
	claims, err := ParseLicense(data, publicKey)
	if err != nil {
		return nil, err
	}
	gatewayID, secret, err := s.identity()
	if err != nil {
		return nil, fmt.Errorf("load gateway identity: %w", err)
	}
	if err := MatchGateway(claims, gatewayID, secret); err != nil {
		return nil, err
	}
	return claims, nil
}
