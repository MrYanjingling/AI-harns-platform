package options

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNextLicenseGuardTime(t *testing.T) {
	location := time.FixedZone("test", 8*60*60)
	tests := []struct {
		now  time.Time
		want time.Time
	}{
		{
			now:  time.Date(2026, 11, 18, 10, 4, 59, 0, location),
			want: time.Date(2026, 11, 18, 10, 5, 0, 0, location),
		},
		{
			now:  time.Date(2026, 11, 18, 10, 5, 0, 0, location),
			want: time.Date(2026, 11, 18, 11, 5, 0, 0, location),
		},
		{
			now:  time.Date(2026, 11, 18, 10, 6, 0, 0, location),
			want: time.Date(2026, 11, 18, 11, 5, 0, 0, location),
		},
	}
	for _, test := range tests {
		assert.Equal(t, test.want, nextLicenseGuardTime(test.now))
	}
}
