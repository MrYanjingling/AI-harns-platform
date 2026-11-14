package licensemgr

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// LoadPublicKey reads and parses a PEM encoded RSA public key.
func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("%w: public key file is not configured", ErrPublicKeyUnavailable)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: read public key: %v", ErrPublicKeyUnavailable, err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("%w: invalid public key PEM", ErrPublicKeyUnavailable)
	}
	if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		publicKey, ok := key.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("%w: public key is not RSA", ErrPublicKeyUnavailable)
		}
		return publicKey, nil
	}
	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: parse public key: %v", ErrPublicKeyUnavailable, err)
	}
	return publicKey, nil
}

// LoadPrivateKey reads and parses a PEM encoded RSA private key.
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid private key PEM")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		privateKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not RSA")
		}
		return privateKey, nil
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	return privateKey, nil
}

// SignLicense signs claims with the license issuer's RSA private key.
func SignLicense(privateKey *rsa.PrivateKey, claims *Claims) (string, error) {
	if privateKey == nil {
		return "", fmt.Errorf("private key is nil")
	}
	if claims == nil {
		return "", fmt.Errorf("%w: claims are required", ErrInvalid)
	}
	now := time.Now().UTC()
	claims.Issuer = Issuer
	if claims.IssuedAt == nil {
		claims.IssuedAt = jwt.NewNumericDate(now)
	}
	if err := validateShape(claims); err != nil {
		return "", err
	}
	if err := validateTimes(claims, now); err != nil {
		return "", err
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("sign license: %w", err)
	}
	return signed, nil
}

// ParseLicense verifies a license JWT and returns its claims.
func ParseLicense(data string, publicKey *rsa.PublicKey) (*Claims, error) {
	data = strings.TrimSpace(data)
	if data == "" {
		return nil, fmt.Errorf("%w: license data is empty", ErrInvalid)
	}
	if publicKey == nil {
		return nil, fmt.Errorf("%w: public key is nil", ErrPublicKeyUnavailable)
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(data, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, fmt.Errorf("%w: unexpected signing method %q", ErrInvalid, token.Method.Alg())
		}
		return publicKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}), jwt.WithoutClaimsValidation())
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if token == nil || !token.Valid {
		return nil, fmt.Errorf("%w: signature verification failed", ErrInvalid)
	}
	if err := validateShape(claims); err != nil {
		return nil, err
	}
	return claims, nil
}

// MatchGateway checks whether the license belongs to the current gateway.
func MatchGateway(claims *Claims, gatewayID, secret string) error {
	if claims == nil {
		return fmt.Errorf("%w: claims are required", ErrInvalid)
	}
	if !secureEqual(claims.GatewayID, gatewayID) || !secureEqual(claims.Secret, secret) {
		return ErrGatewayMismatch
	}
	return nil
}

// ValidateForImport rejects an already expired license or software deadline.
func ValidateForImport(claims *Claims, now time.Time) error {
	if claims == nil || claims.ExpiresAt == nil || claims.ExpiresAt.Time.IsZero() || claims.SoftwareRunDeadline.IsZero() {
		return fmt.Errorf("%w: license times are required", ErrInvalid)
	}
	return validateTimes(claims, now)
}

func validateShape(claims *Claims) error {
	switch {
	case claims.Issuer != Issuer:
		return fmt.Errorf("%w: invalid issuer", ErrInvalid)
	case claims.ExpiresAt == nil || claims.ExpiresAt.Time.IsZero():
		return fmt.Errorf("%w: license expiration is required", ErrInvalid)
	case strings.TrimSpace(claims.GatewayID) == "":
		return fmt.Errorf("%w: gatewayId is required", ErrInvalid)
	case strings.TrimSpace(claims.Secret) == "":
		return fmt.Errorf("%w: secret is required", ErrInvalid)
	case claims.MaxVariable <= 0:
		return fmt.Errorf("%w: maxVariable must be greater than 0", ErrInvalid)
	case strings.TrimSpace(claims.Contact) == "":
		return fmt.Errorf("%w: contact is required", ErrInvalid)
	case strings.TrimSpace(claims.ContactEmail) == "":
		return fmt.Errorf("%w: contactEmail is required", ErrInvalid)
	case strings.TrimSpace(claims.Company) == "":
		return fmt.Errorf("%w: company is required", ErrInvalid)
	case claims.SoftwareRunDeadline.IsZero():
		return fmt.Errorf("%w: softwareRunDeadline is required", ErrInvalid)
	}
	address, err := mail.ParseAddress(strings.TrimSpace(claims.ContactEmail))
	if err != nil || address.Address != strings.TrimSpace(claims.ContactEmail) {
		return fmt.Errorf("%w: contactEmail is invalid", ErrInvalid)
	}
	return nil
}

func validateTimes(claims *Claims, now time.Time) error {
	now = now.UTC()
	if !now.Before(claims.ExpiresAt.Time.UTC()) {
		return ErrExpired
	}
	if !now.Before(claims.SoftwareRunDeadline.UTC()) {
		return ErrExpired
	}
	return nil
}

func secureEqual(left, right string) bool {
	leftHash := sha256.Sum256([]byte(left))
	rightHash := sha256.Sum256([]byte(right))
	return subtle.ConstantTimeCompare(leftHash[:], rightHash[:]) == 1
}

func claimsToInfo(claims *Claims) *Info {
	if claims == nil {
		return nil
	}
	return &Info{
		MaxVariable:         claims.MaxVariable,
		Contact:             claims.Contact,
		ContactEmail:        claims.ContactEmail,
		Company:             claims.Company,
		LicenseExpiresAt:    claims.ExpiresAt.Time.UTC().Format(time.RFC3339),
		SoftwareRunDeadline: claims.SoftwareRunDeadline.UTC().Format(time.RFC3339),
	}
}
