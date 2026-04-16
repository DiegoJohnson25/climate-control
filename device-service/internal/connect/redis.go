package connect

import (
	"github.com/redis/go-redis/v9"
)

func Redis(password string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: password,
		DB:       0,
	})
}
