// Package filecache provides a file cache
package filecache

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
)

var (
	// ErrNoCachedKey is returned when there is no cached key available.
	ErrNoCachedKey = errors.New("no cached key")
)

// FileCache provides a simple file-based cache implementation.
type FileCache struct {
	filesystem afero.Fs
	prefix     string
	cacheDir   string
}

// New creates a new FileCache instance.
func New(
	cacheDir string,
	prefix string,
	filesystem afero.Fs,
) *FileCache {
	return &FileCache{
		filesystem: filesystem,
		prefix:     prefix,
		cacheDir:   cacheDir,
	}
}

// GetFilename constructs the full path for a given cache key.
func (fc *FileCache) GetFilename(id string) string {
	return filepath.Join(fc.cacheDir, fc.prefix, id)
}

// Get retrieves data from the cache for a given key.
func (fc *FileCache) Get(key string) ([]byte, error) {
	filename := fc.GetFilename(key)

	exists, err := afero.Exists(fc.filesystem, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to check if cache file exists %s: %w", filename, err)
	}

	if !exists {
		return nil, ErrNoCachedKey
	}

	data, err := afero.ReadFile(fc.filesystem, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file %s: %w", filename, err)
	}

	return data, nil
}

// Set stores data in the cache for a given key.
//
//nolint:mnd
func (fc *FileCache) Set(key string, data []byte) error {
	filename := fc.GetFilename(key)
	if err := fc.filesystem.MkdirAll(filepath.Dir(filename), 0o750); err != nil {
		return fmt.Errorf("failed to create cache directory %s: %w", filepath.Dir(filename), err)
	}

	err := afero.WriteFile(fc.filesystem, filename, data, 0o600)
	if err != nil {
		return fmt.Errorf("failed to write cache file %s: %w", filename, err)
	}

	return nil
}

// Delete removes the cache entry for a given key.
func (fc *FileCache) Delete(key string) error {
	filename := fc.GetFilename(key)
	if exists, err := afero.Exists(fc.filesystem, filename); err != nil {
		return fmt.Errorf("failed to check if cache file exists %s: %w", filename, err)
	} else if exists {
		err := fc.filesystem.Remove(filename)
		if err != nil {
			return fmt.Errorf("failed to delete cache file %s: %w", filename, err)
		}
	}

	return nil
}
