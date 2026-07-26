package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"deploymate/internal/model"
)

type Cache interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Del(ctx context.Context, key string) error
	Close() error
}

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisCache(addr, password string, db int) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &RedisCache{
		client: client,
		ttl:    5 * time.Second,
	}
}

func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *RedisCache) Del(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *RedisCache) GetDesiredState(ctx context.Context, orgID, deploymentID string) (*model.DeploymentSpec, bool) {
	key := c.desiredStateKey(orgID, deploymentID)
	var spec model.DeploymentSpec
	if err := c.Get(ctx, key, &spec); err != nil {
		return nil, false
	}
	return &spec, true
}

func (c *RedisCache) SetDesiredState(ctx context.Context, orgID, deploymentID string, spec *model.DeploymentSpec) error {
	key := c.desiredStateKey(orgID, deploymentID)
	return c.Set(ctx, key, spec, c.ttl)
}

func (c *RedisCache) InvalidateDesiredState(ctx context.Context, orgID, deploymentID string) error {
	key := c.desiredStateKey(orgID, deploymentID)
	return c.Del(ctx, key)
}

func (c *RedisCache) desiredStateKey(orgID, deploymentID string) string {
	return "deploymate:desired:" + orgID + ":" + deploymentID
}

func (c *RedisCache) Close() error {
	return c.client.Close()
}
