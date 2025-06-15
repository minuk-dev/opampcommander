//nolint:goconst
package filecache_test

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/filecache"
)

func TestFileCache_SetAndGet(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	cacheDir := "test_cache"
	prefix := "test_prefix"
	cache := filecache.New(cacheDir, prefix, fs)

	key := "test_key"
	data := []byte("test_data")

	// Test Set
	err := cache.Set(key, data)
	require.NoError(t, err, "Failed to set cache")

	// Test Get
	retrievedData, err := cache.Get(key)
	require.NoError(t, err)
	assert.Equal(t, data, retrievedData)
}

func TestFileCache_Delete(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	cacheDir := "test_cache"
	prefix := "test_prefix"
	cache := filecache.New(cacheDir, prefix, fs)

	key := "test_key"
	data := []byte("test_data")

	// Set data
	err := cache.Set(key, data)
	require.NoError(t, err, "Failed to set cache")

	// Delete data
	err = cache.Delete(key)
	require.NoError(t, err, "Failed to delete cache")

	// Verify deletion
	_, err = cache.Get(key)
	assert.Error(t, err, "Expected error when getting deleted key")
}

func TestFileCache_NonExistentKey(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	cacheDir := "test_cache"
	prefix := "test_prefix"
	fc := filecache.New(cacheDir, prefix, fs)

	key := "non_existent_key"

	// Test Get for non-existent key
	_, err := fc.Get(key)
	assert.Error(t, err, "Expected error for non-existent key")
}
