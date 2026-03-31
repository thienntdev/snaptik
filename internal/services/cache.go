package services

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/log"
	"time"
	"github.com/redis/go-redis/v9"
	"github.com/thienntdev/snaptiktok/internal/config"
	"github.com/thienntdev/snaptiktok/internal/models"
)

// CacheService handles Redis caching operations
type CacheService struct {
	client *redis.Client
	ttl    time.Duration
	ctx    context.Context
}

// NewCacheService creates a new Redis cache service
func NewCacheService(cfg *config.Config) *CacheService {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctx := context.Background()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("⚠️ Redis connection failed: %v (cache disabled, using in-memory fallback)", err)
	} else {
		log.Info("✅ Redis connected successfully")
	}

	return &CacheService{
		client: client,
		ttl:    cfg.CacheTTL,
		ctx:    ctx,
	}
}

// cacheKey generates a consistent cache key for a URL
func (s *CacheService) cacheKey(url string) string {
	return fmt.Sprintf("snaptiktok:video:%s", url)
}

// Get retrieves cached video info by URL
func (s *CacheService) Get(url string) (*models.VideoInfo, error) {
	key := s.cacheKey(url)
	data, err := s.client.Get(s.ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var info models.VideoInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// Set stores video info in cache with TTL
func (s *CacheService) Set(url string, info *models.VideoInfo) error {
	key := s.cacheKey(url)
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}

	return s.client.Set(s.ctx, key, data, s.ttl).Err()
}

// IncrementDailyCounter increments a daily download counter
func (s *CacheService) IncrementDailyCounter() (int64, error) {
	key := fmt.Sprintf("snaptiktok:stats:downloads:%s", time.Now().Format("2006-01-02"))
	val, err := s.client.Incr(s.ctx, key).Result()
	if err != nil {
		return 0, err
	}
	// Set expiry on the key (48 hours to allow overlap)
	s.client.Expire(s.ctx, key, 48*time.Hour)
	return val, nil
}

// GetDailyCounter gets today's download count
func (s *CacheService) GetDailyCounter() (int64, error) {
	key := fmt.Sprintf("snaptiktok:stats:downloads:%s", time.Now().Format("2006-01-02"))
	val, err := s.client.Get(s.ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

// Close closes the Redis connection
func (s *CacheService) Close() error {
	return s.client.Close()
}
