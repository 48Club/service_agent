package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

type server struct {
	*redis.Client
}

func (s server) SaveCache(k string, v interface{}, life time.Duration) error {
	cmd := s.Client.Set(ctx, k, v, life)
	return cmd.Err()
}

func (s server) Keys(pattern string) ([]string, error) {
	cmd := s.Client.Keys(ctx, pattern)
	return cmd.Result()
}

func (s server) GetCache(k string) (string, error) {
	cmd := s.Client.Get(ctx, k)
	return cmd.Result()
}

func (s server) Del(k string) error {
	cmd := s.Client.Del(ctx, k)
	return cmd.Err()
}

func New(db int) server {
	return server{redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       db,
	})}
}
