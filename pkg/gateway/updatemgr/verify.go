package updatemgr

import (
	"archive/zip"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/mod/semver"

	"harns-platform/pkg/gateway/licensemgr"
)

func loadPublicKey(file string) (*rsa.PublicKey, error) {
	key, err := licensemgr.LoadPublicKey(file)
	if err != nil {
		return nil, fmt.Errorf("load OTA public key: %w", err)
	}
	return key, nil
}

func verifyManifestSignature(publicKey *rsa.PublicKey, manifest []byte, signature string) error {
	signature = strings.TrimSpace(signature)
	if signature == "" {
		return fmt.Errorf("manifest signature is empty")
	}
	decoded, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("decode manifest signature: %w", err)
	}
	hash := sha256.Sum256(manifest)
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hash[:], decoded); err != nil {
		return fmt.Errorf("manifest signature verification failed")
	}
	return nil
}

func signatureURL(manifestURL string) string {
	parsed, err := url.Parse(manifestURL)
	if err != nil {
		return manifestURL + ".sig"
	}
	parsed.Path += ".sig"
	return parsed.String()
}

func validateDownloadURL(raw string, allowedHosts []string, allowHTTP bool) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("invalid download URL: %w", err)
	}
	if parsed.Scheme != "https" && !(allowHTTP && parsed.Scheme == "http") {
		return nil, fmt.Errorf("download URL scheme %q is not allowed", parsed.Scheme)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("download URL host is empty")
	}
	if !hostAllowed(parsed, allowedHosts) {
		return nil, fmt.Errorf("download host %q is not allowed", parsed.Host)
	}
	return parsed, nil
}

func hostAllowed(parsed *url.URL, allowedHosts []string) bool {
	if len(allowedHosts) == 0 {
		return true
	}
	host := strings.ToLower(parsed.Host)
	hostname := strings.ToLower(parsed.Hostname())
	for _, item := range allowedHosts {
		item = strings.TrimSpace(strings.ToLower(item))
		if item == "" {
			continue
		}
		if host == item || hostname == item {
			return true
		}
	}
	return false
}

func resolvePackageURL(manifestURL, packagePath string) (string, error) {
	base, err := url.Parse(manifestURL)
	if err != nil {
		return "", fmt.Errorf("parse manifest URL: %w", err)
	}
	relative, err := url.Parse(packagePath)
	if err != nil {
		return "", fmt.Errorf("parse package path: %w", err)
	}
	return base.ResolveReference(relative).String(), nil
}

func versionWithPrefix(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "v") {
		return value
	}
	return "v" + value
}

func isNewerVersion(candidate, installed string) (bool, error) {
	candidate = versionWithPrefix(candidate)
	installed = versionWithPrefix(installed)
	if !semver.IsValid(candidate) {
		return false, fmt.Errorf("invalid candidate version %q", candidate)
	}
	if !semver.IsValid(installed) {
		return false, fmt.Errorf("invalid installed version %q", installed)
	}
	return semver.Compare(candidate, installed) > 0, nil
}

func validateManifest(manifest *Manifest) error {
	if manifest.Format != 1 {
		return fmt.Errorf("unsupported manifest format %d", manifest.Format)
	}
	if manifest.APIVersion != 1 {
		return fmt.Errorf("unsupported manifest apiVersion %d", manifest.APIVersion)
	}
	if manifest.Product != "harns-gateway" {
		return fmt.Errorf("unexpected product %q", manifest.Product)
	}
	if manifest.ReleaseVersion == "" {
		return fmt.Errorf("manifest releaseVersion is required")
	}
	if manifest.OS != runtime.GOOS || manifest.Arch != runtime.GOARCH {
		return fmt.Errorf("release platform %s/%s does not match %s/%s",
			manifest.OS, manifest.Arch, runtime.GOOS, runtime.GOARCH)
	}
	if manifest.Backend.SHA256 == "" {
		return fmt.Errorf("manifest backend sha256 is required")
	}
	if manifest.Frontend.SHA256 == "" {
		return fmt.Errorf("manifest frontend sha256 is required")
	}
	if manifest.Package.Path == "" || manifest.Package.SHA256 == "" {
		return fmt.Errorf("manifest package path and sha256 are required")
	}
	return nil
}

func sha256File(file string) (string, error) {
	handle, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer handle.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, handle); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func extractPackage(archivePath, destination string, maxBytes int64) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open package: %w", err)
	}
	defer reader.Close()

	if err := os.MkdirAll(destination, 0o755); err != nil {
		return fmt.Errorf("create package staging directory: %w", err)
	}
	var total int64
	for _, file := range reader.File {
		name := path.Clean(strings.TrimPrefix(file.Name, "/"))
		if name == "." || name == "" {
			continue
		}
		if name == ".." || strings.HasPrefix(name, "../") || strings.Contains(name, "/../") {
			return fmt.Errorf("unsafe package path %q", file.Name)
		}
		if !strings.HasPrefix(name, "payload/") {
			continue
		}
		target := filepath.Join(destination, filepath.FromSlash(name))
		relative, err := filepath.Rel(destination, target)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return fmt.Errorf("unsafe package target %q", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("create package directory: %w", err)
			}
			continue
		}
		if !file.Mode().IsRegular() {
			return fmt.Errorf("non-regular package entry %q is not allowed", file.Name)
		}
		if maxBytes > 0 && total+int64(file.UncompressedSize64) > maxBytes {
			return fmt.Errorf("package expanded size exceeds limit")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create package parent directory: %w", err)
		}
		if err := extractZipFile(file, target); err != nil {
			return err
		}
		total += int64(file.UncompressedSize64)
	}
	return nil
}

func extractZipFile(file *zip.File, target string) error {
	source, err := file.Open()
	if err != nil {
		return fmt.Errorf("open package entry %q: %w", file.Name, err)
	}
	defer source.Close()
	mode := file.Mode().Perm()
	if mode == 0 {
		mode = 0o644
	}
	targetFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("create package entry %q: %w", file.Name, err)
	}
	_, copyErr := io.Copy(targetFile, source)
	closeErr := targetFile.Close()
	if copyErr != nil {
		return fmt.Errorf("write package entry %q: %w", file.Name, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close package entry %q: %w", file.Name, closeErr)
	}
	return nil
}
