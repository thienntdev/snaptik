package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	// Server
	Port    string
	AppName string
	BaseURL string

	// Redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// Cache
	CacheTTL time.Duration

	// Rate Limiting
	RateLimitMax    int
	RateLimitWindow time.Duration

	// Temp Storage
	TempDir      string
	CleanupInterval time.Duration
	FileMaxAge   time.Duration

	// Environment
	IsProd bool
}

// Load reads configuration from environment variables with sensible defaults
func Load() *Config {
	return &Config{
		Port:    getEnv("PORT", "3000"),
		AppName: getEnv("APP_NAME", "SnapTiktok"),
		BaseURL: getEnv("BASE_URL", "https://snaptiktok.com"),

		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		CacheTTL: time.Duration(getEnvInt("CACHE_TTL_HOURS", 3)) * time.Hour,

		RateLimitMax:    getEnvInt("RATE_LIMIT_MAX", 15),
		RateLimitWindow: time.Duration(getEnvInt("RATE_LIMIT_WINDOW_SEC", 60)) * time.Second,

		TempDir:         getEnv("TEMP_DIR", "./tmp/downloads"),
		CleanupInterval: 10 * time.Minute,
		FileMaxAge:      1 * time.Hour,

		IsProd: getEnv("ENV", "development") == "production",
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}
