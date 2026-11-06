package things

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"sync"
	"time"
)

const (
	defaultAuthDenyCacheTTL        = time.Hour
	defaultAuthDenyCacheMaxEntries = 50000
)

// AuthDenyCacheConfig controls the local cache for unknown MQTT credentials.
type AuthDenyCacheConfig struct {
	TTL        time.Duration
	MaxEntries int
}

type authDenyCache struct {
	mu      sync.Mutex
	ttl     time.Duration
	max     int
	now     func() time.Time
	entries map[string]time.Time
}

func newAuthDenyCache(cfg AuthDenyCacheConfig) *authDenyCache {
	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = defaultAuthDenyCacheTTL
	}
	maxEntries := cfg.MaxEntries
	if maxEntries <= 0 {
		maxEntries = defaultAuthDenyCacheMaxEntries
	}
	return &authDenyCache{
		ttl:     ttl,
		max:     maxEntries,
		now:     time.Now,
		entries: make(map[string]time.Time),
	}
}

func authDenyCacheKey(clientID, username, password string) string {
	hash := sha256.New()
	var size [8]byte
	for _, value := range []string{clientID, username, password} {
		binary.BigEndian.PutUint64(size[:], uint64(len(value)))
		hash.Write(size[:])
		hash.Write([]byte(value))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func (c *authDenyCache) contains(key string) bool {
	if c == nil || key == "" {
		return false
	}
	now := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()
	expiresAt, ok := c.entries[key]
	if !ok {
		return false
	}
	if !now.Before(expiresAt) {
		delete(c.entries, key)
		return false
	}
	return true
}

func (c *authDenyCache) add(key string) {
	if c == nil || key == "" {
		return
	}
	now := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= c.max {
		for existing := range c.entries {
			delete(c.entries, existing)
			break
		}
	}
	c.entries[key] = now.Add(c.ttl)
}
