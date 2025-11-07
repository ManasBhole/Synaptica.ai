package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Server
	ServerPort     string
	ServerHost     string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	MaxRequestBody int64

	// Database
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	PostgresSSLMode  string

	// Redis
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	// Kafka
	KafkaBrokers []string
	KafkaGroupID string

	// ClickHouse
	ClickHouseHost     string
	ClickHousePort     string
	ClickHouseUser     string
	ClickHousePassword string
	ClickHouseDB       string

	// OIDC
	OIDCIssuer       string
	OIDCClientID     string
	OIDCClientSecret string

	// LLM
	LLMAPIKey    string
	LLMBaseURL   string
	LLMModelName string

	// Feature Store
	FeatureStoreCacheTTL time.Duration

	// Gateway specific
	IngestionBaseURL      string
	GatewayRequestTimeout time.Duration
	GatewayRateLimitRPS   int
	GatewayRateLimitBurst int
}

func Load() *Config {
	return &Config{
		ServerPort:     getEnv("SERVER_PORT", "8080"),
		ServerHost:     getEnv("SERVER_HOST", "0.0.0.0"),
		ReadTimeout:    getDuration("READ_TIMEOUT", 30*time.Second),
		WriteTimeout:   getDuration("WRITE_TIMEOUT", 30*time.Second),
		MaxRequestBody: int64(getIntEnv("MAX_REQUEST_BODY_BYTES", 4*1024*1024)),

		PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:     getEnv("POSTGRES_USER", "synaptica"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", "synaptica123"),
		PostgresDB:       getEnv("POSTGRES_DB", "synaptica"),
		PostgresSSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getIntEnv("REDIS_DB", 0),

		KafkaBrokers: getStringSliceEnv("KAFKA_BROKERS", []string{"localhost:9092"}),
		KafkaGroupID: getEnv("KAFKA_GROUP_ID", "synaptica-platform"),

		ClickHouseHost:     getEnv("CLICKHOUSE_HOST", "localhost"),
		ClickHousePort:     getEnv("CLICKHOUSE_PORT", "9000"),
		ClickHouseUser:     getEnv("CLICKHOUSE_USER", "default"),
		ClickHousePassword: getEnv("CLICKHOUSE_PASSWORD", ""),
		ClickHouseDB:       getEnv("CLICKHOUSE_DB", "synaptica"),

		OIDCIssuer:       getEnv("OIDC_ISSUER", ""),
		OIDCClientID:     getEnv("OIDC_CLIENT_ID", ""),
		OIDCClientSecret: getEnv("OIDC_CLIENT_SECRET", ""),

		LLMAPIKey:    getEnv("LLM_API_KEY", ""),
		LLMBaseURL:   getEnv("LLM_BASE_URL", "https://api.openai.com/v1"),
		LLMModelName: getEnv("LLM_MODEL_NAME", "gpt-4"),

		FeatureStoreCacheTTL: getDuration("FEATURE_STORE_CACHE_TTL", 5*time.Minute),

		IngestionBaseURL:      getEnv("INGESTION_BASE_URL", "http://localhost:8081"),
		GatewayRequestTimeout: getDuration("GATEWAY_REQUEST_TIMEOUT", 10*time.Second),
		GatewayRateLimitRPS:   getIntEnv("GATEWAY_RATE_LIMIT_RPS", 50),
		GatewayRateLimitBurst: getIntEnv("GATEWAY_RATE_LIMIT_BURST", 100),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getStringSliceEnv(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return []string{value}
	}
	return defaultValue
}

func getDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
