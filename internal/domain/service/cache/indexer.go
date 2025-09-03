// Package cache provides a generic caching mechanism with indexing capabilities.
package cache

import (
	"errors"

	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ port.Indexer[any] = (*Cache[any])(nil)

var (
	ErrNotImplemented = errors.New("not implemented")
)

// KeyFunc is a function that takes an object of type T and returns a string key and an error if any.
type KeyFunc[T any] func(obj T) (string, error)

// KeyError represents an error that occurred while generating a key for an object of type T.
type KeyError[T any] struct {
	Obj T
	Err error
}

// Error implements the error interface.
func (e KeyError[T]) Error() string {
	return e.Err.Error()
}

// Cache is a generic struct that provides caching functionality for objects of type T.
type Cache[T any] struct {
	storage Storage[T]
	keyFunc KeyFunc[T]
}

// NewIndexer creates a new instance of Cache with the provided storage and key function.
func NewIndexer[T any](
	store Storage[T],
	keyFunc KeyFunc[T],
) *Cache[T] {
	return &Cache[T]{
		storage: store,
		keyFunc: keyFunc,
	}
}

// Add adds a new item to the cache.
func (c *Cache[T]) Add(obj T) error {
	key, err := c.keyFunc(obj)
	if err != nil {
		return KeyError[T]{
			Obj: obj,
			Err: err,
		}
	}

	c.storage.Add(key, obj)

	return nil
}

// Update updates an existing item in the cache.
func (c *Cache[T]) Update(obj T) error {
	key, err := c.keyFunc(obj)
	if err != nil {
		return KeyError[T]{
			Obj: obj,
			Err: err,
		}
	}

	c.storage.Update(key, obj)

	return nil
}

// Delete removes an item from the cache by its object.
func (c *Cache[T]) Delete(obj T) error {
	key, err := c.keyFunc(obj)
	if err != nil {
		return KeyError[T]{
			Obj: obj,
			Err: err,
		}
	}

	c.storage.Delete(key)

	return nil
}

// List returns a list of all items in the cache.
func (c *Cache[T]) List() []T {
	return c.storage.List()
}

// ListKeys returns a list of all keys in the cache.
func (c *Cache[T]) ListKeys() []string {
	return c.storage.ListKeys()
}

// GetIndexers returns the indexers used by the cache.
func (c *Cache[T]) GetIndexers() port.Indexers[T] {
	return c.storage.GetIndexers()
}

// Index returns the indexed values for a specific object and index function.
//
//nolint:wrapcheck
func (c *Cache[T]) Index(indexName string, obj T) ([]any, error) {
	return c.storage.Index(indexName, obj)
}

// IndexKeys lists all keys that match the given indexed value for a specific index function.
//
//nolint:wrapcheck
func (c *Cache[T]) IndexKeys(indexName, indexedValue string) ([]string, error) {
	return c.storage.IndexKeys(indexName, indexedValue)
}

// ListIndexFuncValues lists all indexed values for a given index function.
func (c *Cache[T]) ListIndexFuncValues(indexName string) []string {
	return c.storage.ListIndexFuncValues(indexName)
}

// ByIndex retrieves all items that match the given indexed value for a specific index function.
//
//nolint:wrapcheck
func (c *Cache[T]) ByIndex(indexName, indexedValue string) ([]T, error) {
	return c.storage.ByIndex(indexName, indexedValue)
}

// AddIndexers adds new indexers to the cache.
//
//nolint:wrapcheck
func (c *Cache[T]) AddIndexers(newIndexers port.Indexers[T]) error {
	return c.storage.AddIndexers(newIndexers)
}

// Get retrieves an item from the cache by its object.
//
//nolint:nonamedreturns
func (c *Cache[T]) Get(obj T) (item *T, exists bool, err error) {
	key, err := c.keyFunc(obj)
	if err != nil {
		return nil, false, KeyError[T]{
			Obj: obj,
			Err: err,
		}
	}

	return c.GetByKey(key)
}

// GetByKey retrieves an item from the cache by its key.
//
//nolint:nonamedreturns
func (c *Cache[T]) GetByKey(key string) (item *T, exists bool, err error) {
	item, exists = c.storage.Get(key)

	return item, exists, nil
}

// Replace replaces the entire contents of the cache with the provided list of items.
func (c *Cache[T]) Replace(list []T, resourceVersion string) error {
	items := make(map[string]T, len(list))
	for _, item := range list {
		key, err := c.keyFunc(item)
		if err != nil {
			return KeyError[T]{item, err}
		}

		items[key] = item
	}

	c.storage.Replace(items, resourceVersion)

	return nil
}
