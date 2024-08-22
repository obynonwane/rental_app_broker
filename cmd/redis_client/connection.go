package redis_client

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient() (*redis.Client, error) {
	redisAddress := fmt.Sprintf("%s:6379", os.Getenv("REDIS_URL"))
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddress,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("could not connect to Redis: %w", err)
	}

	log.Println("Redis connection successful on broker-service....")

	return rdb, nil
}
