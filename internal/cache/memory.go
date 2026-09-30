// Package cache 提供统一的缓存抽象层，支持内存缓存和 Redis 两种后端实现。
package cache

import (
	"sync"
	"time"
)

// entry 缓存条目，保存值及其过期时间（过期时间为零值表示永不过期）。
type entry struct {
	value      string
	expiration time.Time
}

// isExpired 判断条目是否已过期。
func (e *entry) isExpired() bool {
	return !e.expiration.IsZero() && time.Now().After(e.expiration)
}

// MemoryCache 基于内存的并发安全缓存，适用于单机或 Redis 不可用时的降级场景。
type MemoryCache struct {
	mu      sync.RWMutex
	entries map[string]*entry
}

// NewMemoryCache 创建内存缓存实例，并启动后台过期清理协程。
func NewMemoryCache() *MemoryCache {
	mc := &MemoryCache{
		entries: make(map[string]*entry),
	}
	go mc.cleanupLoop()
	return mc
}

// Set 写入键值，expiration 为存活时长（<=0 表示不过期）。
func (m *MemoryCache) Set(key string, value string, expiration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var exp time.Time
	if expiration > 0 {
		exp = time.Now().Add(expiration)
	}
	m.entries[key] = &entry{value: value, expiration: exp}
}

// Get 读取键对应的值，第二个返回值表示是否存在且未过期。
func (m *MemoryCache) Get(key string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.entries[key]
	if !ok || e.isExpired() {
		return "", false
	}
	return e.value, true
}

// Delete 删除指定键。
func (m *MemoryCache) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, key)
}

// Exists 判断键是否存在且未过期。
func (m *MemoryCache) Exists(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.entries[key]
	return ok && !e.isExpired()
}

// Size 返回当前未过期条目数量。
func (m *MemoryCache) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, e := range m.entries {
		if !e.isExpired() {
			count++
		}
	}
	return count
}

// cleanupLoop 周期性触发过期条目清理，每 5 分钟一次。
func (m *MemoryCache) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		m.cleanup()
	}
}

// cleanup 遍历并删除所有已过期的条目。
func (m *MemoryCache) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for k, e := range m.entries {
		if !e.expiration.IsZero() && now.After(e.expiration) {
			delete(m.entries, k)
		}
	}
}
