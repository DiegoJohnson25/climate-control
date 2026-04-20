// Package initializers holds legacy connection constructors for api-service.
// New connection code should live under internal/connect (see the cleanup
// note in CLAUDE.md).
package initializers

import (
	"github.com/redis/go-redis/v9"
)

// ConnectRedis constructs a go-redis client pointing at the internal Redis instance.
func ConnectRedis(password string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: password,
		DB:       0,
	})
}
