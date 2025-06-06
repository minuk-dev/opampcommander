// Package filecache provides a file cache
package filecache

import (
	"path/filepath"

	"github.com/spf13/afero"
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
	data, err := afero.ReadFile(fc.filesystem, filename)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Set stores data in the cache for a given key.
func (fc *FileCache) Set(key string, data []byte) error {
	filename := fc.GetFilename(key)
	if err := fc.filesystem.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return err
	}
	return afero.WriteFile(fc.filesystem, filename, data, 0o600)
}

// Delete removes the cache entry for a given key.
func (fc *FileCache) Delete(key string) error {
	filename := fc.GetFilename(key)
	if exists, err := afero.Exists(fc.filesystem, filename); err != nil {
		return err
	} else if exists {
		return fc.filesystem.Remove(filename)
	}
	return nil
}
