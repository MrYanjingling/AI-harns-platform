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
