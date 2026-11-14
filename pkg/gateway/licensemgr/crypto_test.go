package licensemgr

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testKeyOnce sync.Once
	testKey     *rsa.PrivateKey
	testKeyErr  error
)

func privateKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	testKeyOnce.Do(func() {
		testKey, testKeyErr = rsa.GenerateKey(rand.Reader, 2048)
	})
	require.NoError(t, testKeyErr)
	return testKey
}

func publicKeyFile(t *testing.T) string {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(&privateKey(t).PublicKey)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "license-public.pem")
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: der,
	}), 0o600))
	return path
}

func validClaims(maxVariable int) *Claims {
	now := time.Now().UTC()
	return &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
		},
		GatewayID:           "gateway-1",
		Secret:              "secret-1",
		MaxVariable:         maxVariable,
		Contact:             "Alice",
		ContactEmail:        "alice@example.com",
		Company:             "ACME",
		SoftwareRunDeadline: now.Add(48 * time.Hour),
	}
}

func signedLicense(t *testing.T, claims *Claims) string {
	t.Helper()
	license, err := SignLicense(privateKey(t), claims)
	require.NoError(t, err)
	return license
}

func TestSignAndParseLicense(t *testing.T) {
	data := signedLicense(t, validClaims(100))
	key, err := LoadPublicKey(publicKeyFile(t))
	require.NoError(t, err)
	claims, err := ParseLicense(data, key)
	require.NoError(t, err)
	assert.Equal(t, Issuer, claims.Issuer)
	assert.Equal(t, "gateway-1", claims.GatewayID)
	assert.Equal(t, "secret-1", claims.Secret)
	assert.Equal(t, 100, claims.MaxVariable)
	assert.Equal(t, "Alice", claims.Contact)
	assert.Equal(t, "alice@example.com", claims.ContactEmail)
	assert.Equal(t, "ACME", claims.Company)
	assert.NotNil(t, claims.ExpiresAt)
	assert.False(t, claims.SoftwareRunDeadline.IsZero())
}

func TestParseLicenseRejectsTamperedData(t *testing.T) {
	data := signedLicense(t, validClaims(100))
	key, err := LoadPublicKey(publicKeyFile(t))
	require.NoError(t, err)
	_, err = ParseLicense(data+"x", key)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalid))
}

func TestParseLicenseRejectsMissingFields(t *testing.T) {
	claims := validClaims(100)
	claims.Company = ""
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Claims.(*Claims).Issuer = Issuer
	data, err := token.SignedString(privateKey(t))
	require.NoError(t, err)
	key, err := LoadPublicKey(publicKeyFile(t))
	require.NoError(t, err)
	_, err = ParseLicense(data, key)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalid))
}

func TestValidateForImportRejectsExpiredClaims(t *testing.T) {
	claims := validClaims(100)
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().UTC().Add(-time.Minute))
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Claims.(*Claims).Issuer = Issuer
	data, err := token.SignedString(privateKey(t))
	require.NoError(t, err)
	key, err := LoadPublicKey(publicKeyFile(t))
	require.NoError(t, err)
	parsed, err := ParseLicense(data, key)
	require.NoError(t, err)
	assert.True(t, errors.Is(ValidateForImport(parsed, time.Now().UTC()), ErrExpired))
}

func TestValidateForImportRejectsPassedSoftwareDeadline(t *testing.T) {
	claims := validClaims(100)
	claims.SoftwareRunDeadline = time.Now().UTC().Add(-time.Minute)
	assert.True(t, errors.Is(ValidateForImport(claims, time.Now().UTC()), ErrExpired))
}

func TestMatchGateway(t *testing.T) {
	claims := validClaims(100)
	assert.NoError(t, MatchGateway(claims, "gateway-1", "secret-1"))
	assert.True(t, errors.Is(MatchGateway(claims, "gateway-2", "secret-1"), ErrGatewayMismatch))
	assert.True(t, errors.Is(MatchGateway(claims, "gateway-1", "secret-2"), ErrGatewayMismatch))
}

func TestLoadPublicKeyPKCS1(t *testing.T) {
	der := x509.MarshalPKCS1PublicKey(&privateKey(t).PublicKey)
	path := filepath.Join(t.TempDir(), "license-public-rsa.pem")
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: der,
	}), 0o600))
	key, err := LoadPublicKey(path)
	require.NoError(t, err)
	assert.Equal(t, privateKey(t).PublicKey.N, key.N)
}

func TestLoadPublicKeyUnavailable(t *testing.T) {
	_, err := LoadPublicKey("")
	assert.True(t, errors.Is(err, ErrPublicKeyUnavailable))
	_, err = LoadPublicKey(filepath.Join(t.TempDir(), "missing.pem"))
	assert.True(t, errors.Is(err, ErrPublicKeyUnavailable))
}
