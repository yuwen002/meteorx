package cache

import (
	"context"
	"errors"
	"time"

	"meteorx/pkg/logger"

	"github.com/redis/go-redis/v9"
)

// ErrRedisUnavailable Redis 不可用时返回的哨兵错误，调用方可据此降级
var ErrRedisUnavailable = errors.New("redis is not available")

type Redis struct {
	Client *redis.Client
}

// NewRedis 创建 Redis 客户端并尝试连接。
// 若连接失败，返回的 Redis 内部 Client 为 nil，上层业务可调用 IsAvailable 判断。
func NewRedis(addr, password string, db int) (*Redis, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		// 连接失败，关闭这个无用的 client，避免泄漏连接
		_ = rdb.Close()
		return &Redis{}, err
	}

	logger.Info("Redis connected successfully")
	return &Redis{Client: rdb}, nil
}

// IsAvailable 判断 Redis 是否可用
func (r *Redis) IsAvailable() bool {
	return r != nil && r.Client != nil
}

// Ping 检查 Redis 连接是否正常
func (r *Redis) Ping(ctx context.Context) error {
	if !r.IsAvailable() {
		return ErrRedisUnavailable
	}
	return r.Client.Ping(ctx).Err()
}

func (r *Redis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if !r.IsAvailable() {
		return ErrRedisUnavailable
	}
	return r.Client.Set(ctx, key, value, expiration).Err()
}

func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	if !r.IsAvailable() {
		return "", ErrRedisUnavailable
	}
	return r.Client.Get(ctx, key).Result()
}

func (r *Redis) Delete(ctx context.Context, key string) error {
	if !r.IsAvailable() {
		return ErrRedisUnavailable
	}
	return r.Client.Del(ctx, key).Err()
}

func (r *Redis) Exists(ctx context.Context, key string) (bool, error) {
	if !r.IsAvailable() {
		return false, ErrRedisUnavailable
	}
	result, err := r.Client.Exists(ctx, key).Result()
	return result > 0, err
}

func (r *Redis) Close() error {
	if r == nil || r.Client == nil {
		return nil
	}
	return r.Client.Close()
}
