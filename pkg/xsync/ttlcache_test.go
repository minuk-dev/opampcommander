package xsync_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/xsync"
)

func TestTTLCache_SetAndGet(t *testing.T) {
	t.Parallel()

	cache := xsync.NewTTLCache[string, int](time.Minute)

	cache.Set("key1", 100)
	cache.Set("key2", 200)

	val, ok := cache.Get("key1")
	require.True(t, ok)
	assert.Equal(t, 100, val)

	val, ok = cache.Get("key2")
	require.True(t, ok)
	assert.Equal(t, 200, val)
}

func TestTTLCache_GetNotFound(t *testing.T) {
	t.Parallel()

	cache := xsync.NewTTLCache[string, int](time.Minute)

	val, ok := cache.Get("nonexistent")
	assert.False(t, ok)
	assert.Equal(t, 0, val)
}

func TestTTLCache_Expiration(t *testing.T) {
	t.Parallel()

	now := time.Now()
	cache := xsync.NewTTLCache[string, string](30 * time.Second)
	cache.SetNowFunc(func() time.Time { return now })

	cache.Set("key", "value")

	// Should be found before expiration
	val, ok := cache.Get("key")
	require.True(t, ok)
	assert.Equal(t, "value", val)

	// Advance time past TTL
	cache.SetNowFunc(func() time.Time { return now.Add(31 * time.Second) })

	// Should not be found after expiration
	val, ok = cache.Get("key")
	assert.False(t, ok)
	assert.Equal(t, "", val)
}

func TestTTLCache_SetWithTTL(t *testing.T) {
	t.Parallel()

	now := time.Now()
	cache := xsync.NewTTLCache[string, string](time.Minute)
	cache.SetNowFunc(func() time.Time { return now })

	// Set with custom shorter TTL
	cache.SetWithTTL("short", "value", 10*time.Second)
	cache.Set("normal", "value")

	// Both should exist
	_, ok := cache.Get("short")
	require.True(t, ok)
	_, ok = cache.Get("normal")
	require.True(t, ok)

	// Advance time past short TTL but before normal TTL
	cache.SetNowFunc(func() time.Time { return now.Add(15 * time.Second) })

	// Short should be expired, normal should exist
	_, ok = cache.Get("short")
	assert.False(t, ok)
	_, ok = cache.Get("normal")
	assert.True(t, ok)
}

func TestTTLCache_Delete(t *testing.T) {
	t.Parallel()

	cache := xsync.NewTTLCache[string, int](time.Minute)

	cache.Set("key", 100)

	val, ok := cache.Get("key")
	require.True(t, ok)
	assert.Equal(t, 100, val)

	cache.Delete("key")

	val, ok = cache.Get("key")
	assert.False(t, ok)
	assert.Equal(t, 0, val)
}

func TestTTLCache_Clear(t *testing.T) {
	t.Parallel()

	cache := xsync.NewTTLCache[string, int](time.Minute)

	cache.Set("key1", 100)
	cache.Set("key2", 200)
	cache.Set("key3", 300)

	assert.Equal(t, 3, cache.Len())

	cache.Clear()

	assert.Equal(t, 0, cache.Len())

	_, ok := cache.Get("key1")
	assert.False(t, ok)
}

func TestTTLCache_Cleanup(t *testing.T) {
	t.Parallel()

	now := time.Now()
	cache := xsync.NewTTLCache[string, string](30 * time.Second)
	cache.SetNowFunc(func() time.Time { return now })

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	// Advance time past TTL
	cache.SetNowFunc(func() time.Time { return now.Add(31 * time.Second) })

	// Add a new non-expired item
	cache.Set("key3", "value3")

	// Before cleanup, length includes expired items
	assert.Equal(t, 3, cache.Len())

	// Cleanup expired items
	cache.Cleanup()

	// After cleanup, only non-expired item remains
	assert.Equal(t, 1, cache.Len())

	_, ok := cache.Get("key1")
	assert.False(t, ok)
	_, ok = cache.Get("key2")
	assert.False(t, ok)
	val, ok := cache.Get("key3")
	assert.True(t, ok)
	assert.Equal(t, "value3", val)
}

func TestTTLCache_GetOrSet(t *testing.T) {
	t.Parallel()

	cache := xsync.NewTTLCache[string, int](time.Minute)

	loadCount := 0
	loader := func() (int, error) {
		loadCount++
		return 42, nil
	}

	// First call should invoke loader
	val, err := cache.GetOrSet("key", loader)
	require.NoError(t, err)
	assert.Equal(t, 42, val)
	assert.Equal(t, 1, loadCount)

	// Second call should use cached value
	val, err = cache.GetOrSet("key", loader)
	require.NoError(t, err)
	assert.Equal(t, 42, val)
	assert.Equal(t, 1, loadCount) // Loader not called again
}

func TestTTLCache_GetOrSet_Error(t *testing.T) {
	t.Parallel()

	cache := xsync.NewTTLCache[string, int](time.Minute)

	expectedErr := errors.New("load error")
	loader := func() (int, error) {
		return 0, expectedErr
	}

	val, err := cache.GetOrSet("key", loader)
	assert.ErrorIs(t, err, expectedErr)
	assert.Equal(t, 0, val)

	// Should not be cached on error
	_, ok := cache.Get("key")
	assert.False(t, ok)
}

func TestTTLCache_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	cache := xsync.NewTTLCache[int, int](time.Minute)

	const goroutines = 100
	const iterations = 100

	done := make(chan bool, goroutines)

	for i := range goroutines {
		go func(id int) {
			for j := range iterations {
				key := (id*iterations + j) % 50 // Use limited key space for contention
				cache.Set(key, id*1000+j)
				_, _ = cache.Get(key)
			}
			done <- true
		}(i)
	}

	for range goroutines {
		<-done
	}

	// If we reach here without deadlock or race condition, the test passes
	assert.True(t, cache.Len() > 0)
}

func TestTTLCache_UpdateExistingKey(t *testing.T) {
	t.Parallel()

	cache := xsync.NewTTLCache[string, int](time.Minute)

	cache.Set("key", 100)

	val, ok := cache.Get("key")
	require.True(t, ok)
	assert.Equal(t, 100, val)

	cache.Set("key", 200)

	val, ok = cache.Get("key")
	require.True(t, ok)
	assert.Equal(t, 200, val)
}

func TestTTLCache_StructValue(t *testing.T) {
	t.Parallel()

	type User struct {
		ID   int
		Name string
	}

	cache := xsync.NewTTLCache[string, *User](time.Minute)

	user := &User{ID: 1, Name: "Alice"}
	cache.Set("user:1", user)

	cached, ok := cache.Get("user:1")
	require.True(t, ok)
	assert.Equal(t, user.ID, cached.ID)
	assert.Equal(t, user.Name, cached.Name)
}
