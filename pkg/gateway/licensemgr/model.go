package licensemgr

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// LicenseRecordID is the primary key of the single persisted license row.
	LicenseRecordID = "license"
	// Issuer identifies licenses generated for Harns Gateway.
	Issuer = "harns-license"

	// StatusUnauthorized means no valid license is loaded.
	StatusUnauthorized = "unauthorized"
	// StatusAuthorized means a valid license is loaded.
	StatusAuthorized = "authorized"

	// DefaultTrialMaxVariable is the variable quota when no license is installed.
	DefaultTrialMaxVariable = 100
	// DefaultTrialDuration is the software run window from gateway initialization.
	DefaultTrialDuration = 30 * 24 * time.Hour
)

var (
	ErrInvalid              = errors.New("license invalid")
	ErrGatewayMismatch      = errors.New("license gateway mismatch")
	ErrExpired              = errors.New("license expired")
	ErrPublicKeyUnavailable = errors.New("license public key unavailable")
)

// Claims is the signed JWT payload of a license.
type Claims struct {
	jwt.RegisteredClaims
	GatewayID           string    `json:"gatewayId"`
	Secret              string    `json:"secret"`
	MaxVariable         int       `json:"maxVariable"`
	Contact             string    `json:"contact"`
	ContactEmail        string    `json:"contactEmail"`
	Company             string    `json:"company"`
	SoftwareRunDeadline time.Time `json:"softwareRunDeadline"`
}

// Info is the license data returned by the query API.
type Info struct {
	MaxVariable         int    `json:"maxVariable"`
	Contact             string `json:"contact"`
	ContactEmail        string `json:"contactEmail"`
	Company             string `json:"company"`
	LicenseExpiresAt    string `json:"licenseExpiresAt"`
	SoftwareRunDeadline string `json:"softwareRunDeadline"`
}

// Status is the response of the license query and import APIs.
type Status struct {
	Status  string `json:"status"`
	License *Info  `json:"license,omitempty"`
}

// ImportRequest is the import API request body.
type ImportRequest struct {
	Data string `json:"data"`
}
