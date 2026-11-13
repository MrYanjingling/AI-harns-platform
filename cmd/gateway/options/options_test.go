package options

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestGatewayFlagsMatchPlatform(t *testing.T) {
	opts := NewDefaultOptions()
	fs := pflag.NewFlagSet("harns-gateway", pflag.ContinueOnError)
	opts.AddFlags(fs)

	for _, name := range []string{"port", "graceful-timeout", "cert-file", "key-file"} {
		assert.NotNil(t, fs.Lookup(name), "missing flag %s", name)
	}
}

func TestAuthEnabledFromEnv(t *testing.T) {
	t.Setenv(_envAuthEnabled, "")
	assert.False(t, authEnabledFromEnv())
	t.Setenv(_envAuthEnabled, "true")
	assert.True(t, authEnabledFromEnv())
	t.Setenv(_envAuthEnabled, "false")
	assert.False(t, authEnabledFromEnv())
	t.Setenv(_envAuthEnabled, "invalid")
	assert.False(t, authEnabledFromEnv())
}
