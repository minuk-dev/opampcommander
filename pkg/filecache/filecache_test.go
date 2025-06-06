package filecache_test

import (
	"testing"

	"github.com/minuk-dev/opampcommander/pkg/filecache"
	"github.com/spf13/afero"
)

func TestFileCache_SetAndGet(t *testing.T) {
	fs := afero.NewMemMapFs()
	cacheDir := "test_cache"
	prefix := "test_prefix"
	fc := filecache.New(cacheDir, prefix, fs)

	key := "test_key"
	data := []byte("test_data")

	// Test Set
	if err := fc.Set(key, data); err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Test Get
	retrievedData, err := fc.Get(key)
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}

	if string(retrievedData) != string(data) {
		t.Errorf("Expected %s, got %s", string(data), string(retrievedData))
	}
}

func TestFileCache_Delete(t *testing.T) {
	fs := afero.NewMemMapFs()
	cacheDir := "test_cache"
	prefix := "test_prefix"
	fc := filecache.New(cacheDir, prefix, fs)

	key := "test_key"
	data := []byte("test_data")

	// Set data
	if err := fc.Set(key, data); err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Delete data
	if err := fc.Delete(key); err != nil {
		t.Fatalf("Failed to delete cache: %v", err)
	}

	// Verify deletion
	_, err := fc.Get(key)
	if err == nil {
		t.Errorf("Expected error for non-existent key, got nil")
	}
}

func TestFileCache_NonExistentKey(t *testing.T) {
	fs := afero.NewMemMapFs()
	cacheDir := "test_cache"
	prefix := "test_prefix"
	fc := filecache.New(cacheDir, prefix, fs)

	key := "non_existent_key"

	// Test Get for non-existent key
	_, err := fc.Get(key)
	if err == nil {
		t.Errorf("Expected error for non-existent key, got nil")
	}
}
