package gatewaymgr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatGB(t *testing.T) {
	assert.Equal(t, "1.5GB", formatGB(1610612736))
	assert.Equal(t, "0.0GB", formatGB(0))
}
