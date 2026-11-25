package updatemgr

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsNewerVersion(t *testing.T) {
	newer, err := isNewerVersion("3.1.1", "3.1.0")
	require.NoError(t, err)
	assert.True(t, newer)

	newer, err = isNewerVersion("3.1.0", "3.1.0")
	require.NoError(t, err)
	assert.False(t, newer)

	newer, err = isNewerVersion("3.0.9", "3.1.0")
	require.NoError(t, err)
	assert.False(t, newer)

	_, err = isNewerVersion("invalid", "3.1.0")
	require.Error(t, err)
}

func TestValidateDownloadURL(t *testing.T) {
	allowed := []string{"minio.internal:9000"}
	_, err := validateDownloadURL("http://minio.internal:9000/a.zip", allowed, false)
	require.Error(t, err)
	_, err = validateDownloadURL("http://minio.internal:9000/a.zip", allowed, true)
	require.NoError(t, err)
	_, err = validateDownloadURL("https://minio.internal:9000/a.zip", allowed, false)
	require.NoError(t, err)
	_, err = validateDownloadURL("https://example.com/a.zip", allowed, false)
	require.Error(t, err)
}

func TestStateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "updates", "state.json")
	want := &State{
		Installed: Installed{ReleaseVersion: "3.1.0", BackendVersion: "3.1.0"},
		Task:      &Task{TaskID: "task-1", State: StateReady},
	}
	require.NoError(t, saveState(path, want))
	got, err := loadState(path)
	require.NoError(t, err)
	assert.Equal(t, want.Installed, got.Installed)
	require.NotNil(t, got.Task)
	assert.Equal(t, "task-1", got.Task.TaskID)

	_, err = os.Stat(path)
	require.NoError(t, err)
}
