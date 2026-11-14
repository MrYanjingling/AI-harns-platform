package options

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	baseoptions "harns-platform/pkg/generic/options"
)

func TestGatewayFlagsMatchPlatform(t *testing.T) {
	opts := NewDefaultOptions()
	fs := pflag.NewFlagSet("harns-gateway", pflag.ContinueOnError)
	opts.AddFlags(fs)

	for _, name := range []string{"port", "auth-enabled", "graceful-timeout", "cert-file", "key-file", "license-public-key-file"} {
		assert.NotNil(t, fs.Lookup(name), "missing flag %s", name)
	}
}

func TestResolveAuthEnabled(t *testing.T) {
	t.Setenv(_envAuthEnabled, "")
	assert.True(t, resolveAuthEnabled(true))
	assert.False(t, resolveAuthEnabled(false))
	t.Setenv(_envAuthEnabled, "true")
	assert.True(t, resolveAuthEnabled(false))
	t.Setenv(_envAuthEnabled, "false")
	assert.False(t, resolveAuthEnabled(true))
	t.Setenv(_envAuthEnabled, "invalid")
	assert.True(t, resolveAuthEnabled(true))
	assert.False(t, resolveAuthEnabled(false))
}

func TestConfigFileAuthEnabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("auth-enabled: true\nlicense:\n  public-key-file: /tmp/license.pem\n"), 0o600))
	opts := NewDefaultOptions()
	opts.ConfigFile = path
	require.NoError(t, baseoptions.ParseAndApplyConfigFile(opts, nil))
	assert.True(t, opts.AuthEnabled)
	assert.Equal(t, "/tmp/license.pem", opts.License.PublicKeyFile)
}
