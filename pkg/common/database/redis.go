package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/synaptica-ai/platform/pkg/common/config"
	"github.com/synaptica-ai/platform/pkg/common/logger"
)

var (
	redisClient *redis.Client
	redisOnce   sync.Once
)

func GetRedis() *redis.Client {
	redisOnce.Do(func() {
		cfg := config.Load()
		redisClient = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := redisClient.Ping(ctx).Err(); err != nil {
			logger.Log.WithError(err).Error("Failed to connect to Redis")
		} else {
			logger.Log.Info("Connected to Redis")
		}
	})

	return redisClient
}

func CloseRedis() error {
	if redisClient != nil {
		return redisClient.Close()
	}
	return nil
}

