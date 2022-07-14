package rediscache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
)

type Client struct {
	redis *redis.Client
}

func NewRedisClient(c *redis.Client) *Client {
	return &Client{
		redis: c,
	}
}

func (c *Client) SetValue(ctx context.Context, k string, v interface{}, expiration time.Duration) error {
	valBts, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, k, valBts, expiration).Err()
}

func (c *Client) GetValue(ctx context.Context, k string, dest interface{}) error {
	bts, err := c.redis.Get(ctx, k).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(bts, &dest)
}
