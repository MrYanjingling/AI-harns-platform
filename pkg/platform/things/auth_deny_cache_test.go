package things

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAuthDenyCacheKeyHidesPassword(t *testing.T) {
	key := authDenyCacheKey("client-1", "user-1", "top-secret")
	assert.NotContains(t, key, "top-secret")
	assert.NotContains(t, key, "user-1")
	assert.NotEqual(t, key, authDenyCacheKey("client-1", "user-1", "other-secret"))
}

func TestAuthDenyCacheExpiresEntries(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	cache := newAuthDenyCache(AuthDenyCacheConfig{TTL: time.Hour, MaxEntries: 10})
	cache.now = func() time.Time { return now }

	cache.add("key-1")
	assert.True(t, cache.contains("key-1"))

	now = now.Add(time.Hour)
	assert.False(t, cache.contains("key-1"))
}

func TestAuthDenyCacheBoundsEntries(t *testing.T) {
	cache := newAuthDenyCache(AuthDenyCacheConfig{TTL: time.Hour, MaxEntries: 2})
	cache.add("key-1")
	cache.add("key-2")
	cache.add("key-3")

	assert.Len(t, cache.entries, 2)
}
