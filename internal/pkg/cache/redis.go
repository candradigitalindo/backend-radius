package cache

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"

	"github.com/candrasyahputra/radius-server/internal/config"
)

func Connect(cfg *config.Config) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Host + ":" + cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Unable to connect to Redis: %v", err)
	}

	return client
}
