package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Repository struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewRepository(rdb *redis.Client, ttlDays int) *Repository {
	return &Repository{
		rdb: rdb,
		ttl: time.Duration(ttlDays) * 24 * time.Hour,
	}
}

func (r *Repository) Store(ctx context.Context, token string, userID uuid.UUID) error {
	return r.rdb.Set(ctx, hashToken(token), userID.String(), r.ttl).Err()
}

func (r *Repository) GetUserID(ctx context.Context, token string) (uuid.UUID, error) {
	val, err := r.rdb.Get(ctx, hashToken(token)).Result()
	if err == redis.Nil {
		return uuid.UUID{}, ErrTokenNotFound
	}
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("redis get: %w", err)
	}

	id, err := uuid.Parse(val)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("parse user id: %w", err)
	}
	return id, nil
}

func (r *Repository) Delete(ctx context.Context, token string) error {
	n, err := r.rdb.Del(ctx, hashToken(token)).Result()
	if err != nil {
		return fmt.Errorf("redis del: %w", err)
	}
	if n == 0 {
		return ErrTokenNotFound
	}
	return nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
