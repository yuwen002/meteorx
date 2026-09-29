package cache

import (
	"sync"
	"time"
)

type entry struct {
	value      string
	expiration time.Time
}

func (e *entry) isExpired() bool {
	return !e.expiration.IsZero() && time.Now().After(e.expiration)
}

type MemoryCache struct {
	mu      sync.RWMutex
	entries map[string]*entry
}

func NewMemoryCache() *MemoryCache {
	mc := &MemoryCache{
		entries: make(map[string]*entry),
	}
	go mc.cleanupLoop()
	return mc
}

func (m *MemoryCache) Set(key string, value string, expiration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var exp time.Time
	if expiration > 0 {
		exp = time.Now().Add(expiration)
	}
	m.entries[key] = &entry{value: value, expiration: exp}
}

func (m *MemoryCache) Get(key string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.entries[key]
	if !ok || e.isExpired() {
		return "", false
	}
	return e.value, true
}

func (m *MemoryCache) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, key)
}

func (m *MemoryCache) Exists(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.entries[key]
	return ok && !e.isExpired()
}

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

func (m *MemoryCache) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		m.cleanup()
	}
}

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