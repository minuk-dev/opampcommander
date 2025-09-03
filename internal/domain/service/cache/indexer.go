package cache

import (
	"errors"

	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ port.Indexer[any] = (*cache[any])(nil)

var (
	ErrNotImplemented = errors.New("not implemented")
)

type KeyFunc[T any] func(obj T) (string, error)

type KeyError[T any] struct {
	Obj T
	Err error
}

func (e KeyError[T]) Error() string {
	return e.Err.Error()
}

type cache[T any] struct {
	storage Storage[T]
	keyFunc KeyFunc[T]
}

func NewIndexer[T any](
	store Storage[T],
	keyFunc KeyFunc[T],
) *cache[T] {
	return &cache[T]{
		storage: store,
		keyFunc: keyFunc,
	}
}

func (c *cache[T]) Add(obj T) error {
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

func (c *cache[T]) Update(obj T) error {
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

func (c *cache[T]) Delete(obj T) error {
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

func (c *cache[T]) List() []T {
	return c.storage.List()
}

func (c *cache[T]) ListKeys() []string {
	return c.storage.ListKeys()
}

func (c *cache[T]) GetIndexers() port.Indexers[T] {
	return c.storage.GetIndexers()
}

//nolint:wrapcheck
func (c *cache[T]) Index(indexName string, obj T) ([]any, error) {
	return c.storage.Index(indexName, obj)
}

//nolint:wrapcheck
func (c *cache[T]) IndexKeys(indexName, indexedValue string) ([]string, error) {
	return c.storage.IndexKeys(indexName, indexedValue)
}

func (c *cache[T]) ListIndexFuncValues(indexName string) []string {
	return c.storage.ListIndexFuncValues(indexName)
}

//nolint:wrapcheck
func (c *cache[T]) ByIndex(indexName, indexedValue string) ([]T, error) {
	return c.storage.ByIndex(indexName, indexedValue)
}

//nolint:wrapcheck
func (c *cache[T]) AddIndexers(newIndexers port.Indexers[T]) error {
	return c.storage.AddIndexers(newIndexers)
}

func (c *cache[T]) Get(obj T) (item *T, exists bool, err error) {
	key, err := c.keyFunc(obj)
	if err != nil {
		return nil, false, KeyError[T]{
			Obj: obj,
			Err: err,
		}
	}

	return c.GetByKey(key)
}

func (c *cache[T]) GetByKey(key string) (item *T, exists bool, err error) {
	item, exists = c.storage.Get(key)

	return item, exists, nil
}

func (c *cache[T]) Replace(list []T, resourceVersion string) error {
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
