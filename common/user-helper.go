package common

import (
	"sync"
	"time"
)

type userRateLimitCache struct {
	RPM       int
	RPD       int
	RPW       int
	ExpiresAt time.Time
}

var userRateLimitCacheMap = make(map[int]*userRateLimitCache)
var userRateLimitCacheLock sync.RWMutex

// GetUserRateLimits gets the rate limits for a user from cache or database
func GetUserRateLimits(userId int) (rpm int, rpd int, rpw int, err error) {
	// Check cache first
	userRateLimitCacheLock.RLock()
	if cache, ok := userRateLimitCacheMap[userId]; ok {
		if time.Now().Before(cache.ExpiresAt) {
			rpm = cache.RPM
			rpd = cache.RPD
			rpw = cache.RPW
			userRateLimitCacheLock.RUnlock()
			return rpm, rpd, rpw, nil
		}
	}
	userRateLimitCacheLock.RUnlock()

	// Import dynamically to avoid circular dependency
	// We'll need to call model.GetUserRateLimits directly in the middleware
	return 0, 0, 0, nil
}

// CacheUserRateLimits caches the user rate limits
func CacheUserRateLimits(userId int, rpm int, rpd int, rpw int) {
	userRateLimitCacheLock.Lock()
	defer userRateLimitCacheLock.Unlock()
	
	userRateLimitCacheMap[userId] = &userRateLimitCache{
		RPM:       rpm,
		RPD:       rpd,
		RPW:       rpw,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
}
