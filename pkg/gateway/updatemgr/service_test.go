package updatemgr

import (
	"archive/zip"
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/sqlite"
)

func TestAcceptTaskDownloadsAndVerifiesRelease(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	publicKey := &privateKey.PublicKey
	root := t.TempDir()
	publicKeyPath := filepath.Join(root, "ota-public.pem")
	publicDER, err := x509.MarshalPKIXPublicKey(publicKey)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(publicKeyPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER}), 0o600))

	packageBytes := makePackage(t)
	packageHash := sha256.Sum256(packageBytes)
	backendBytes := []byte("new gateway")
	backendHash := sha256.Sum256(backendBytes)
	manifest := Manifest{
		Format:         1,
		APIVersion:     1,
		Product:        "harns-gateway",
		ReleaseVersion: "3.1.1",
		OS:             runtime.GOOS,
		Arch:           runtime.GOARCH,
		Backend:        ManifestPart{Version: "3.1.0", SHA256: hex.EncodeToString(backendHash[:])},
		Frontend:       ManifestPart{Version: "3.1.1", SHA256: hex.EncodeToString(backendHash[:])},
		Package:        ManifestPackage{Path: "release.zip", SHA256: hex.EncodeToString(packageHash[:]), Size: int64(len(packageBytes))},
	}
	manifestBytes, err := json.Marshal(manifest)
	require.NoError(t, err)
	digest := sha256.Sum256(manifestBytes)
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest.json":
			_, _ = w.Write(manifestBytes)
		case "/manifest.json.sig":
			_, _ = w.Write([]byte(base64.StdEncoding.EncodeToString(signature)))
		case "/release.zip":
			_, _ = w.Write(packageBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	db, err := sqliteOpenForTest(filepath.Join(root, "gateway.db"))
	require.NoError(t, err)
	defer db.Close()
	service, err := NewService(Options{
		GatewayID:     "gateway-1",
		DB:            db,
		Secret:        "test-secret",
		PublicKeyFile: publicKeyPath,
		StagingDir:    filepath.Join(root, "updates"),
		UIDir:         filepath.Join(root, "ui"),
	})
	require.NoError(t, err)
	service.mu.Lock()
	service.state.Installed = Installed{
		ReleaseVersion:  "3.0.0",
		BackendVersion:  "3.0.0",
		FrontendVersion: "3.0.0",
	}
	require.NoError(t, saveState(service.statePath, service.state))
	service.mu.Unlock()

	service.acceptTask(TaskMessage{
		APIVersion:  1,
		Type:        "gateway.ota.task",
		TaskID:      "task-1",
		GatewayID:   "gateway-1",
		ManifestURL: server.URL + "/manifest.json",
	})

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		state := service.Snapshot()
		if state.Task != nil && state.Task.State == StateReady {
			assert.Equal(t, "3.1.1", state.Task.ReleaseVersion)
			assert.True(t, state.Task.RestartRequired)
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("update task did not become ready: %+v", service.Snapshot().Task)
}

func makePackage(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	files := map[string][]byte{
		"payload/bin/gateway":              []byte("new gateway"),
		"payload/ui/gateway-ui/index.html": []byte("<html></html>"),
	}
	for name, data := range files {
		entry, err := writer.Create(name)
		require.NoError(t, err)
		_, err = entry.Write(data)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	return buffer.Bytes()
}

func sqliteOpenForTest(path string) (*sql.DB, error) {
	return sqlite.Open(path)
}
