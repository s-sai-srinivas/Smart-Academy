package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"coding-platform/config"

	"github.com/redis/go-redis/v9"
)

// CacheService provides a unified caching layer using Redis or in-memory fallback
type CacheService struct {
	ctx         context.Context
	redisClient *redis.Client
	memoryCache *memoryCache
	enabled     bool
}

// memoryCache is an in-memory fallback when Redis is not available
type memoryCache struct {
	items map[string]*cacheItem
	mu    sync.RWMutex
}

type cacheItem struct {
	expiresAt time.Time
	data      []byte
}

// Global cache instance
var cache *CacheService
var cacheOnce sync.Once

// InitCache initializes the global cache service
func InitCache() error {
	var initErr error
	cacheOnce.Do(func() {
		cfg := config.AppConfig
		ctx := context.Background()

		cache = &CacheService{
			enabled: cfg.RedisEnabled,
			ctx:     ctx,
		}

		if cfg.RedisEnabled {
			addr := fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort)
			cache.redisClient = redis.NewClient(&redis.Options{
				Addr:     addr,
				Password: cfg.RedisPassword,
				DB:       cfg.RedisDB,
			})

			// Test connection
			if err := cache.redisClient.Ping(ctx).Err(); err != nil {
				log.Printf("Warning: Redis connection failed: %v. Falling back to in-memory cache", err)
				cache.enabled = false
				initErr = err
			} else {
				log.Printf("Redis cache connected at %s", addr)
			}
		}

		if !cache.enabled {
			cache.memoryCache = &memoryCache{
				items: make(map[string]*cacheItem),
			}
			log.Println("Using in-memory cache (Redis disabled)")
		}
	})
	return initErr
}

// GetCache returns the global cache instance
func GetCache() *CacheService {
	if cache == nil {
		_ = InitCache()
	}
	return cache
}

// Get retrieves a value from cache
func (c *CacheService) Get(key string, result interface{}) bool {
	if !c.enabled {
		return c.getFromMemory(key, result)
	}
	return c.getFromRedis(key, result)
}

// Set stores a value in cache with TTL
func (c *CacheService) Set(key string, value interface{}, ttl time.Duration) bool {
	data, err := json.Marshal(value)
	if err != nil {
		log.Printf("Cache marshal error: %v", err)
		return false
	}

	if !c.enabled {
		return c.setToMemory(key, data, ttl)
	}
	return c.setToRedis(key, data, ttl)
}

// Delete removes a value from cache
func (c *CacheService) Delete(key string) bool {
	if !c.enabled {
		return c.deleteFromMemory(key)
	}
	return c.deleteFromRedis(key)
}

// DeletePattern deletes all keys matching a pattern
func (c *CacheService) DeletePattern(pattern string) bool {
	if !c.enabled {
		return c.deletePatternFromMemory(pattern)
	}
	return c.deletePatternFromRedis(pattern)
}

// Redis operations
func (c *CacheService) getFromRedis(key string, result interface{}) bool {
	data, err := c.redisClient.Get(c.ctx, key).Bytes()
	if err == redis.Nil {
		return false
	} else if err != nil {
		log.Printf("Redis get error: %v", err)
		return false
	}

	if err := json.Unmarshal(data, result); err != nil {
		log.Printf("Cache unmarshal error: %v", err)
		return false
	}
	return true
}

func (c *CacheService) setToRedis(key string, data []byte, ttl time.Duration) bool {
	if err := c.redisClient.Set(c.ctx, key, data, ttl).Err(); err != nil {
		log.Printf("Redis set error: %v", err)
		return false
	}
	return true
}

func (c *CacheService) deleteFromRedis(key string) bool {
	if err := c.redisClient.Del(c.ctx, key).Err(); err != nil {
		log.Printf("Redis delete error: %v", err)
		return false
	}
	return true
}

func (c *CacheService) deletePatternFromRedis(pattern string) bool {
	var cursor uint64
	for {
		keys, nextCursor, err := c.redisClient.Scan(c.ctx, cursor, pattern, 100).Result()
		if err != nil {
			log.Printf("Redis scan error: %v", err)
			return false
		}
		if len(keys) > 0 {
			c.redisClient.Del(c.ctx, keys...)
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return true
}

// Memory operations (fallback)
func (c *CacheService) getFromMemory(key string, result interface{}) bool {
	c.memoryCache.mu.RLock()
	defer c.memoryCache.mu.RUnlock()

	item, exists := c.memoryCache.items[key]
	if !exists {
		return false
	}

	if time.Now().After(item.expiresAt) {
		return false
	}

	if err := json.Unmarshal(item.data, result); err != nil {
		log.Printf("Memory cache unmarshal error: %v", err)
		return false
	}
	return true
}

func (c *CacheService) setToMemory(key string, data []byte, ttl time.Duration) bool {
	c.memoryCache.mu.Lock()
	defer c.memoryCache.mu.Unlock()

	c.memoryCache.items[key] = &cacheItem{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	}
	return true
}

func (c *CacheService) deleteFromMemory(key string) bool {
	c.memoryCache.mu.Lock()
	defer c.memoryCache.mu.Unlock()

	delete(c.memoryCache.items, key)
	return true
}

func (c *CacheService) deletePatternFromMemory(pattern string) bool {
	c.memoryCache.mu.Lock()
	defer c.memoryCache.mu.Unlock()

	// Simple pattern matching for cache keys
	for key := range c.memoryCache.items {
		if matchPattern(key, pattern) {
			delete(c.memoryCache.items, key)
		}
	}
	return true
}

// matchPattern does simple glob-style pattern matching
func matchPattern(key, pattern string) bool {
	// Handle * wildcard
	if pattern == "*" {
		return true
	}
	// Handle prefix* pattern
	if len(pattern) > 1 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(key) >= len(prefix) && key[:len(prefix)] == prefix
	}
	return key == pattern
}

// Cache key helpers for analytics
const (
	DashboardCachePrefix        = "cache:dashboard:"
	SectionAnalyticsCachePrefix = "cache:section_analytics:"
	HODAnalyticsCachePrefix     = "cache:hod_analytics:"
	StudentAnalyticsCachePrefix = "cache:student_analytics:"
	CourseCompletionCachePrefix = "cache:course_completion:"
	FacultyPerfCachePrefix      = "cache:faculty_perf:"
	LeaderboardCachePrefix      = "cache:leaderboard:"
)

// Default TTL values
var (
	DashboardCacheTTL   = 5 * time.Minute
	SectionAnalyticsTTL = 10 * time.Minute
	HODAnalyticsTTL     = 15 * time.Minute
	StudentAnalyticsTTL = 5 * time.Minute
	CourseCompletionTTL = 30 * time.Minute
	FacultyPerfTTL      = 30 * time.Minute
	TopicProficiencyTTL = 10 * time.Minute
	LeaderboardCacheTTL = 2 * time.Minute
)

// InvalidateUserCache invalidates all cached data for a specific user
func InvalidateUserCache(userRegdNo string) {
	c := GetCache()
	if c == nil {
		return
	}
	// Delete user-specific cache entries
	c.DeletePattern(DashboardCachePrefix + userRegdNo + "*")
	c.DeletePattern(StudentAnalyticsCachePrefix + userRegdNo + "*")
}

// InvalidateSectionCache invalidates cache for a specific section
func InvalidateSectionCache(sectionID uint) {
	c := GetCache()
	if c == nil {
		return
	}
	c.DeletePattern(SectionAnalyticsCachePrefix + fmt.Sprintf("%d", sectionID) + "*")
}

// InvalidateCourseCache invalidates cache for a specific course
func InvalidateCourseCache(courseID uint) {
	c := GetCache()
	if c == nil {
		return
	}
	c.DeletePattern(CourseCompletionCachePrefix + fmt.Sprintf("%d", courseID) + "*")
}
