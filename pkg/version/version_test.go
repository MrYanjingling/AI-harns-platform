package version

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionDefaultAndInjected(t *testing.T) {
	original := gitVersion
	t.Cleanup(func() { gitVersion = original })

	gitVersion = ""
	assert.Equal(t, DefaultVersion, Version())

	gitVersion = "$Format:%H$"
	assert.Equal(t, DefaultVersion, Version())

	gitVersion = "v3.1.2"
	assert.Equal(t, "3.1.2", Version())

	gitVersion = "3.1.2-dirty"
	assert.Equal(t, "3.1.2-dirty", Version())
}

func TestCommitIDDoesNotFallbackToVersion(t *testing.T) {
	originalCommit := gitCommit
	originalVersion := gitVersion
	t.Cleanup(func() {
		gitCommit = originalCommit
		gitVersion = originalVersion
	})

	gitCommit = "$Format:%H$"
	gitVersion = "v9.9.9"
	commit := CommitID()
	assert.False(t, strings.Contains(commit, "9.9.9"))
}
