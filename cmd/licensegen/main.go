package main

import (
	"crypto/rsa"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"harns-platform/pkg/gateway/licensemgr"
)

//go:embed dev-private.pem
var devLicensePrivateKeyPEM string

func main() {
	var (
		privateKey          string
		gatewayID           string
		secret              string
		maxVariable         int
		contact             string
		contactEmail        string
		company             string
		licenseExpiresAt    string
		softwareRunDeadline string
		output              string
	)
	flag.StringVar(&privateKey, "private-key", "", "RSA private key PEM file (PKCS8 or PKCS1)")
	flag.StringVar(&gatewayID, "gateway-id", "", "Gateway id")
	flag.StringVar(&secret, "secret", "", "Gateway secret")
	flag.IntVar(&maxVariable, "max-variable", 0, "Maximum variable count")
	flag.StringVar(&contact, "contact", "", "License contact")
	flag.StringVar(&contactEmail, "contact-email", "", "License contact email")
	flag.StringVar(&company, "company", "", "License company")
	flag.StringVar(&licenseExpiresAt, "license-expires-at", "", "License expiration time (RFC3339)")
	flag.StringVar(&softwareRunDeadline, "software-run-deadline", "", "Software run deadline (RFC3339)")
	flag.StringVar(&output, "out", "", "Output file (default stdout)")
	flag.Parse()

	expiresAt, err := parseTime("--license-expires-at", licenseExpiresAt)
	if err != nil {
		fail(err.Error())
	}
	deadline, err := parseTime("--software-run-deadline", softwareRunDeadline)
	if err != nil {
		fail(err.Error())
	}

	var key *rsa.PrivateKey
	if strings.TrimSpace(privateKey) == "" {
		fmt.Fprintln(os.Stderr, "using embedded dev/test license private key")
		key, err = parsePrivateKeyPEM([]byte(devLicensePrivateKeyPEM))
	} else {
		key, err = licensemgr.LoadPrivateKey(privateKey)
	}
	if err != nil {
		fail(err.Error())
	}
	claims := &licensemgr.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		GatewayID:           strings.TrimSpace(gatewayID),
		Secret:              strings.TrimSpace(secret),
		MaxVariable:         maxVariable,
		Contact:             strings.TrimSpace(contact),
		ContactEmail:        strings.TrimSpace(contactEmail),
		Company:             strings.TrimSpace(company),
		SoftwareRunDeadline: deadline,
	}
	license, err := licensemgr.SignLicense(key, claims)
	if err != nil {
		fail(err.Error())
	}

	if strings.TrimSpace(output) == "" {
		fmt.Println(license)
		return
	}
	if err := os.WriteFile(output, []byte(license+"\n"), 0o600); err != nil {
		fail(err.Error())
	}
}

func parsePrivateKeyPEM(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid private key PEM")
	}
	if parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		key, ok := parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not RSA")
		}
		return key, nil
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func parseTime(flagName, value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("%s is required", flagName)
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must use RFC3339: %w", flagName, err)
	}
	return parsed.UTC(), nil
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
