package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMemoryCache_SetAndGet(t *testing.T) {
	mc := NewMemoryCache()

	mc.Set("key1", "value1", 0)
	val, ok := mc.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)
}

func TestMemoryCache_GetMissing(t *testing.T) {
	mc := NewMemoryCache()

	_, ok := mc.Get("nonexistent")
	assert.False(t, ok)
}

func TestMemoryCache_Expiration(t *testing.T) {
	mc := NewMemoryCache()

	mc.Set("temp", "data", 100*time.Millisecond)
	val, ok := mc.Get("temp")
	assert.True(t, ok)
	assert.Equal(t, "data", val)

	time.Sleep(150 * time.Millisecond)
	_, ok = mc.Get("temp")
	assert.False(t, ok, "expired entry should not be returned")
}

func TestMemoryCache_Delete(t *testing.T) {
	mc := NewMemoryCache()

	mc.Set("del-me", "value", 0)
	mc.Delete("del-me")
	_, ok := mc.Get("del-me")
	assert.False(t, ok)
}

func TestMemoryCache_Exists(t *testing.T) {
	mc := NewMemoryCache()

	mc.Set("exists", "yes", 0)
	assert.True(t, mc.Exists("exists"))
	assert.False(t, mc.Exists("nope"))
}

func TestMemoryCache_Size(t *testing.T) {
	mc := NewMemoryCache()

	mc.Set("a", "1", 0)
	mc.Set("b", "2", 0)
	mc.Set("c", "3", 0)
	assert.Equal(t, 3, mc.Size())

	mc.Delete("b")
	assert.Equal(t, 2, mc.Size())
}

func TestMemoryCache_Overwrite(t *testing.T) {
	mc := NewMemoryCache()

	mc.Set("key", "old", 0)
	mc.Set("key", "new", 0)
	val, ok := mc.Get("key")
	assert.True(t, ok)
	assert.Equal(t, "new", val)
}