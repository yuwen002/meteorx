package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"meteorx/pkg/logger"

	"github.com/redis/go-redis/v9"
)

// PermissionCache 权限缓存服务
// 使用 Redis 缓存用户权限，减少数据库查询
type PermissionCache struct {
	redis     *redis.Client
	ttl       time.Duration
	keyPrefix string
}

// NewPermissionCache 创建权限缓存服务
func NewPermissionCache(rdb *redis.Client, ttl time.Duration) *PermissionCache {
	if ttl == 0 {
		ttl = 5 * time.Minute // 默认缓存 5 分钟
	}
	return &PermissionCache{
		redis:     rdb,
		ttl:       ttl,
		keyPrefix: "perm:user:",
	}
}

// GetUserPermissions 获取用户权限（带缓存）
func (pc *PermissionCache) GetUserPermissions(ctx context.Context, userID string, loader func(ctx context.Context, userID string) ([]string, error)) ([]string, error) {
	cacheKey := pc.keyPrefix + userID

	// 尝试从缓存获取
	cached, err := pc.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var permissions []string
		if err := json.Unmarshal([]byte(cached), &permissions); err == nil {
			logger.Debug("Permission cache hit", "user_id", userID)
			return permissions, nil
		}
	}

	// 缓存未命中，从数据库加载
	permissions, err := loader(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 写入缓存
	data, err := json.Marshal(permissions)
	if err == nil {
		if err := pc.redis.Set(ctx, cacheKey, data, pc.ttl).Err(); err != nil {
			logger.Warn("Failed to cache permissions", "user_id", userID, "error", err)
		}
	}

	return permissions, nil
}

// InvalidateUserPermissions 使指定用户的权限缓存失效
func (pc *PermissionCache) InvalidateUserPermissions(ctx context.Context, userID string) error {
	cacheKey := pc.keyPrefix + userID
	return pc.redis.Del(ctx, cacheKey).Err()
}

// InvalidateAllPermissions 使所有用户的权限缓存失效（权限变更时调用）
func (pc *PermissionCache) InvalidateAllPermissions(ctx context.Context) error {
	// 使用 SCAN 遍历所有权限缓存键
	pattern := pc.keyPrefix + "*"
	iter := pc.redis.Scan(ctx, 0, pattern, 100).Iterator()
	count := 0
	for iter.Next(ctx) {
		if err := pc.redis.Del(ctx, iter.Val()).Err(); err != nil {
			logger.Warn("Failed to delete permission cache", "key", iter.Val(), "error", err)
		}
		count++
	}
	if err := iter.Err(); err != nil {
		return err
	}

	logger.Info("Invalidated all permission caches", "count", count)
	return nil
}

// ResourcePermissionCache 资源级权限缓存
type ResourcePermissionCache struct {
	redis     *redis.Client
	ttl       time.Duration
	keyPrefix string
}

// NewResourcePermissionCache 创建资源级权限缓存
func NewResourcePermissionCache(rdb *redis.Client, ttl time.Duration) *ResourcePermissionCache {
	if ttl == 0 {
		ttl = 3 * time.Minute // 资源权限缓存时间较短
	}
	return &ResourcePermissionCache{
		redis:     rdb,
		ttl:       ttl,
		keyPrefix: "perm:resource:",
	}
}

// CheckPermission 检查资源权限（带缓存）
func (rpc *ResourcePermissionCache) CheckPermission(
	ctx context.Context,
	userID, resourceType, resourceID, action string,
	loader func(ctx context.Context, userID, resourceType, resourceID, action string) (bool, error),
) (bool, error) {
	cacheKey := fmt.Sprintf("%s%s:%s:%s:%s", rpc.keyPrefix, userID, resourceType, resourceID, action)

	// 尝试从缓存获取
	cached, err := rpc.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		return cached == "1", nil
	}

	// 缓存未命中，执行检查
	allowed, err := loader(ctx, userID, resourceType, resourceID, action)
	if err != nil {
		return false, err
	}

	// 写入缓存
	value := "0"
	if allowed {
		value = "1"
	}
	if err := rpc.redis.Set(ctx, cacheKey, value, rpc.ttl).Err(); err != nil {
		logger.Warn("Failed to cache resource permission",
			"user_id", userID,
			"resource_id", resourceID,
			"error", err,
		)
	}

	return allowed, nil
}

// InvalidateResourcePermission 使指定资源的权限缓存失效
func (rpc *ResourcePermissionCache) InvalidateResourcePermission(ctx context.Context, resourceType, resourceID string) error {
	pattern := fmt.Sprintf("%s*:%s:%s:*", rpc.keyPrefix, resourceType, resourceID)
	iter := rpc.redis.Scan(ctx, 0, pattern, 100).Iterator()
	count := 0
	for iter.Next(ctx) {
		if err := rpc.redis.Del(ctx, iter.Val()).Err(); err != nil {
			logger.Warn("Failed to delete resource permission cache", "key", iter.Val(), "error", err)
		}
		count++
	}
	return iter.Err()
}