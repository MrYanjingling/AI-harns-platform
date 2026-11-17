package options

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveStorageDirectory(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "data", "gateway.db")
	logPath := filepath.Join(root, "log", "gateway.log")

	assert.Equal(t, filepath.Dir(dbPath), resolveStorageDirectory(dbPath, logPath))
	assert.Equal(t, root, resolveStorageDirectory(filepath.Join(root, "gateway.db"), logPath))
	assert.Equal(t, filepath.Dir(logPath), resolveStorageDirectory("", logPath))
	assert.Equal(t, "", resolveStorageDirectory("", ""))
	assert.Equal(t, "", resolveStorageDirectory(":memory:", ""))
}
