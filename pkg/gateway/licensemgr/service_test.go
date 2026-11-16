package licensemgr

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/sqlite"
)

func testIdentity() (string, string, error) {
	return "gateway-1", "secret-1", nil
}

func newTestService(t *testing.T, publicKeyFile string) (*Service, *Store, *sql.DB) {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)
	require.NoError(t, store.Initialize(context.Background()))
	return NewService(store, testIdentity, publicKeyFile), store, db
}

func TestServiceImportQueryAndLoad(t *testing.T) {
	svc, store, db := newTestService(t, publicKeyFile(t))
	ctx := context.Background()

	assert.Equal(t, StatusUnauthorized, svc.GetStatus().Status)

	data := signedLicense(t, validClaims(100))
	status, err := svc.Import(ctx, "  "+data+"  ")
	require.NoError(t, err)
	require.NotNil(t, status.License)
	assert.Equal(t, StatusAuthorized, status.Status)
	assert.Equal(t, 100, status.License.MaxVariable)
	assert.Equal(t, "Alice", status.License.Contact)

	stored, err := store.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, data, stored)

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(1) FROM license`).Scan(&count))
	assert.Equal(t, 1, count)

	updated := signedLicense(t, validClaims(200))
	_, err = svc.Import(ctx, updated)
	require.NoError(t, err)
	assert.Equal(t, 200, svc.GetStatus().License.MaxVariable)
	require.NoError(t, db.QueryRow(`SELECT COUNT(1) FROM license`).Scan(&count))
	assert.Equal(t, 1, count)

	reloaded := NewService(store, testIdentity, publicKeyFile(t))
	require.NoError(t, reloaded.Load(ctx))
	reloadedStatus := reloaded.GetStatus()
	assert.Equal(t, StatusAuthorized, reloadedStatus.Status)
	assert.Equal(t, 200, reloadedStatus.License.MaxVariable)
}

func TestServiceLoadWithoutLicense(t *testing.T) {
	svc, _, _ := newTestService(t, publicKeyFile(t))
	require.NoError(t, svc.Load(context.Background()))
	status := svc.GetStatus()
	assert.Equal(t, StatusUnauthorized, status.Status)
	require.NotNil(t, status.License)
	assert.Equal(t, DefaultTrialMaxVariable, status.License.MaxVariable)
	assert.NotEmpty(t, status.License.SoftwareRunDeadline)
}

func TestServiceLoadWithoutLicenseUsesTrial(t *testing.T) {
	svc, _, db := newTestService(t, publicKeyFile(t))
	initTime := time.Now().UTC().Add(-time.Hour)
	_, err := db.Exec(`CREATE TABLE gateway (id TEXT PRIMARY KEY, init_time TEXT NOT NULL)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO gateway (id, init_time) VALUES (?, ?)`,
		"gateway-1", initTime.Format(time.RFC3339Nano))
	require.NoError(t, err)

	require.NoError(t, svc.Load(context.Background()))
	status := svc.GetStatus()
	assert.Equal(t, StatusUnauthorized, status.Status)
	require.NotNil(t, status.License)
	assert.Equal(t, DefaultTrialMaxVariable, status.License.MaxVariable)
	assert.Equal(t,
		initTime.Add(DefaultTrialDuration).UTC().Format(time.RFC3339),
		status.License.SoftwareRunDeadline,
	)
}

func TestServiceImportRejectsMismatchedGateway(t *testing.T) {
	svc, _, _ := newTestService(t, publicKeyFile(t))
	claims := validClaims(100)
	claims.GatewayID = "another-gateway"
	_, err := svc.Import(context.Background(), signedLicense(t, claims))
	assert.True(t, errors.Is(err, ErrGatewayMismatch))
}

func TestServiceImportRejectsExpiredLicense(t *testing.T) {
	svc, _, _ := newTestService(t, publicKeyFile(t))
	claims := validClaims(100)
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().UTC().Add(-time.Minute))
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Claims.(*Claims).Issuer = Issuer
	data, err := token.SignedString(privateKey(t))
	require.NoError(t, err)
	_, err = svc.Import(context.Background(), data)
	assert.True(t, errors.Is(err, ErrExpired))
}

func TestServiceImportRejectsMissingPublicKey(t *testing.T) {
	svc, _, _ := newTestService(t, "")
	_, err := svc.Import(context.Background(), signedLicense(t, validClaims(100)))
	assert.True(t, errors.Is(err, ErrPublicKeyUnavailable))
}

func TestServiceImportRejectsEmptyData(t *testing.T) {
	svc, _, _ := newTestService(t, publicKeyFile(t))
	_, err := svc.Import(context.Background(), "   ")
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrInvalid))
}

func TestServiceStatusBecomesUnauthorizedAfterExpiry(t *testing.T) {
	svc, _, _ := newTestService(t, publicKeyFile(t))
	svc.claims = validClaims(100)
	svc.claims.ExpiresAt = jwt.NewNumericDate(time.Now().UTC().Add(-time.Minute))
	status := svc.GetStatus()
	assert.Equal(t, StatusUnauthorized, status.Status)
	require.NotNil(t, status.License)
	assert.Equal(t, DefaultTrialMaxVariable, status.License.MaxVariable)
	assert.NotEmpty(t, status.License.SoftwareRunDeadline)
}
