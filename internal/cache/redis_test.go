package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
)

func newTestRedis(t *testing.T) (*Redis, *miniredis.Miniredis, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	r, err := NewRedis(mr.Addr(), "", 0)
	if err != nil {
		mr.Close()
		t.Fatalf("failed to connect to miniredis: %v", err)
	}
	return r, mr, func() { mr.Close() }
}

func TestNewRedis_ConnectionFailed(t *testing.T) {
	r, err := NewRedis("127.0.0.1:19999", "", 0)
	assert.Error(t, err)
	assert.False(t, r.IsAvailable())
}

func TestIsAvailable_AfterFailedConnection(t *testing.T) {
	r, err := NewRedis("127.0.0.1:19999", "", 0)
	assert.Error(t, err)
	assert.False(t, r.IsAvailable())
}

func TestPing_Unavailable(t *testing.T) {
	r := &Redis{}
	err := r.Ping(context.Background())
	assert.Equal(t, ErrRedisUnavailable, err)
}

func TestSet_Unavailable(t *testing.T) {
	r := &Redis{}
	err := r.Set(context.Background(), "key", "value", time.Minute)
	assert.Equal(t, ErrRedisUnavailable, err)
}

func TestGet_Unavailable(t *testing.T) {
	r := &Redis{}
	val, err := r.Get(context.Background(), "key")
	assert.Equal(t, "", val)
	assert.Equal(t, ErrRedisUnavailable, err)
}

func TestDelete_Unavailable(t *testing.T) {
	r := &Redis{}
	err := r.Delete(context.Background(), "key")
	assert.Equal(t, ErrRedisUnavailable, err)
}

func TestExists_Unavailable(t *testing.T) {
	r := &Redis{}
	exists, err := r.Exists(context.Background(), "key")
	assert.False(t, exists)
	assert.Equal(t, ErrRedisUnavailable, err)
}

func TestSetAndGet(t *testing.T) {
	r, _, cleanup := newTestRedis(t)
	defer cleanup()

	err := r.Set(context.Background(), "hello", "world", time.Minute)
	assert.NoError(t, err)

	val, err := r.Get(context.Background(), "hello")
	assert.NoError(t, err)
	assert.Equal(t, "world", val)
}

func TestGet_KeyNotFound(t *testing.T) {
	r, _, cleanup := newTestRedis(t)
	defer cleanup()

	val, err := r.Get(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Equal(t, "", val)
}

func TestDelete(t *testing.T) {
	r, _, cleanup := newTestRedis(t)
	defer cleanup()

	r.Set(context.Background(), "key", "value", time.Minute)

	err := r.Delete(context.Background(), "key")
	assert.NoError(t, err)

	_, err = r.Get(context.Background(), "key")
	assert.Error(t, err)
}

func TestExists(t *testing.T) {
	r, _, cleanup := newTestRedis(t)
	defer cleanup()

	exists, err := r.Exists(context.Background(), "key")
	assert.NoError(t, err)
	assert.False(t, exists)

	r.Set(context.Background(), "key", "value", time.Minute)

	exists, err = r.Exists(context.Background(), "key")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestPing(t *testing.T) {
	r, _, cleanup := newTestRedis(t)
	defer cleanup()

	err := r.Ping(context.Background())
	assert.NoError(t, err)
}

func TestExpire_KeyExpired(t *testing.T) {
	r, mr, cleanup := newTestRedis(t)
	defer cleanup()

	err := r.Set(context.Background(), "temp", "data", 100*time.Millisecond)
	assert.NoError(t, err)

	val, err := r.Get(context.Background(), "temp")
	assert.NoError(t, err)
	assert.Equal(t, "data", val)

	mr.FastForward(200 * time.Millisecond)

	val, err = r.Get(context.Background(), "temp")
	assert.NotEqual(t, "data", val)
	assert.Error(t, err)
}