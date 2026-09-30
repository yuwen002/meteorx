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

// Redis 封装 go-redis 客户端， Client 为 nil 时表示连接不可用。
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

// Set 写入键值并设置过期时长。
func (r *Redis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if !r.IsAvailable() {
		return ErrRedisUnavailable
	}
	return r.Client.Set(ctx, key, value, expiration).Err()
}

// Get 读取键对应的字符串值。
func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	if !r.IsAvailable() {
		return "", ErrRedisUnavailable
	}
	return r.Client.Get(ctx, key).Result()
}

// Delete 删除指定键。
func (r *Redis) Delete(ctx context.Context, key string) error {
	if !r.IsAvailable() {
		return ErrRedisUnavailable
	}
	return r.Client.Del(ctx, key).Err()
}

// Exists 判断键是否存在。
func (r *Redis) Exists(ctx context.Context, key string) (bool, error) {
	if !r.IsAvailable() {
		return false, ErrRedisUnavailable
	}
	result, err := r.Client.Exists(ctx, key).Result()
	return result > 0, err
}

// Close 关闭 Redis 连接。
func (r *Redis) Close() error {
	if r == nil || r.Client == nil {
		return nil
	}
	return r.Client.Close()
}

// SetNX 仅当键不存在时写入，返回是否设置成功（适用于分布式锁/去重）。
func (r *Redis) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	if !r.IsAvailable() {
		return false, ErrRedisUnavailable
	}
	return r.Client.SetNX(ctx, key, value, expiration).Result()
}

// GetSet 设置新值并返回旧值。
func (r *Redis) GetSet(ctx context.Context, key string, value interface{}) (string, error) {
	if !r.IsAvailable() {
		return "", ErrRedisUnavailable
	}
	return r.Client.GetSet(ctx, key, value).Result()
}

// TTL 返回键的剩余存活时间。
func (r *Redis) TTL(ctx context.Context, key string) (time.Duration, error) {
	if !r.IsAvailable() {
		return 0, ErrRedisUnavailable
	}
	return r.Client.TTL(ctx, key).Result()
}

// Expire 为已有键重新设置过期时长。
func (r *Redis) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if !r.IsAvailable() {
		return ErrRedisUnavailable
	}
	return r.Client.Expire(ctx, key, expiration).Err()
}

// Incr 对键的整数值自增并返回结果（适用于计数器/限流）。
func (r *Redis) Incr(ctx context.Context, key string) (int64, error) {
	if !r.IsAvailable() {
		return 0, ErrRedisUnavailable
	}
	return r.Client.Incr(ctx, key).Result()
}

// DeleteByPattern 扫描并删除匹配指定模式的所有键。
func (r *Redis) DeleteByPattern(ctx context.Context, pattern string) error {
	if !r.IsAvailable() {
		return ErrRedisUnavailable
	}
	iter := r.Client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		if err := r.Client.Del(ctx, iter.Val()).Err(); err != nil {
			logger.Warn("failed to delete cache key", "key", iter.Val(), "error", err)
		}
	}
	return iter.Err()
}
