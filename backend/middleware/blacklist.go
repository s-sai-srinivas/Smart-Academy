package middleware

import (
	"sync"
	"time"
)

// tokenEntry holds the expiry time of a blacklisted token.
type tokenEntry struct {
	expiresAt time.Time
}

// tokenBlacklist is a thread-safe in-memory store for revoked JWT IDs (jti).
//
// NOTE: This implementation is suitable for single-server deployments.
// For multi-server / horizontally-scaled environments, replace the sync.Map
// with a shared Redis store (e.g. github.com/redis/go-redis/v9) so that
// revoked tokens are respected across all nodes.
type tokenBlacklist struct {
	store sync.Map
}

// globalBlacklist is the package-level singleton.
var globalBlacklist = &tokenBlacklist{}

func init() {
	// Start a background goroutine that prunes expired entries every 15 minutes
	// to prevent unbounded memory growth.
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			globalBlacklist.purgeExpired()
		}
	}()
}

// BlacklistToken marks a JWT id (jti) as revoked until exp.
func BlacklistToken(jti string, exp time.Time) {
	globalBlacklist.store.Store(jti, tokenEntry{expiresAt: exp})
}

// IsTokenBlacklisted returns true if the given jti has been explicitly revoked
// and the token has not yet naturally expired.
func IsTokenBlacklisted(jti string) bool {
	v, ok := globalBlacklist.store.Load(jti)
	if !ok {
		return false
	}
	entry := v.(tokenEntry)
	// If the token has already naturally expired, it is no longer dangerous anyway.
	if time.Now().After(entry.expiresAt) {
		globalBlacklist.store.Delete(jti)
		return false
	}
	return true
}

// purgeExpired removes all entries whose natural expiry has passed.
func (b *tokenBlacklist) purgeExpired() {
	now := time.Now()
	b.store.Range(func(key, value any) bool {
		entry := value.(tokenEntry)
		if now.After(entry.expiresAt) {
			b.store.Delete(key)
		}
		return true
	})
}
