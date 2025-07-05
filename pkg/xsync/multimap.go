package xsync

import (
	"strings"
	"sync"
)

// MultiMap is a map that allows multiple keys to point to the same value.
// It is thread-safe and can be used in concurrent environments.
// It is not serializable.
type MultiMap[T any] struct {
	mu sync.RWMutex

	byID        map[string]T
	byIndex     map[string]string
	indexesByID map[string][]string
}

// NewMultiMap creates a new instance of MultiMap.
func NewMultiMap[T any]() *MultiMap[T] {
	return &MultiMap[T]{
		mu:          sync.RWMutex{},
		byID:        make(map[string]T),
		byIndex:     make(map[string]string),
		indexesByID: make(map[string][]string),
	}
}

// Len returns the number of items in the map.
func (m *MultiMap[T]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.byID)
}

// Values returns all values in the map.
func (m *MultiMap[T]) Values() []T {
	m.mu.RLock()
	defer m.mu.RUnlock()

	values := make([]T, 0, len(m.byID))
	for _, value := range m.byID {
		values = append(values, value)
	}

	return values
}

// Keys returns all keys in the map.
func (m *MultiMap[T]) Keys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.byID))
	for key := range m.byID {
		keys = append(keys, key)
	}

	return keys
}

// KeyValues returns all key-value pairs in the map.
func (m *MultiMap[T]) KeyValues() map[string]T {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keyValues := make(map[string]T, len(m.byID))
	for key, value := range m.byID {
		keyValues[key] = value
	}

	return keyValues
}

// Indexes returns all indexes in the map.
func (m *MultiMap[T]) Indexes(indexName string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	indexes := make([]string, 0)

	for indexKey := range m.byIndex {
		if strings.HasPrefix(indexKey, indexName) {
			indexes = append(indexes, indexKey)
		}
	}

	return indexes
}

// Store stores the value in the map with the given key.
// It overwrites the existing value if the key already exists.
func (m *MultiMap[T]) Store(key string, value T, opts ...StoreOption) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.byID[key] = value

	for _, opt := range opts {
		switch opt := opt.(type) {
		case *WithIndexImpl:
			m.byIndex[opt.indexKey] = key
			m.indexesByID[key] = append(m.indexesByID[key], opt.indexKey)
		default:
			panic("unknown option")
		}
	}
}

// AddIndex adds an index to the map for the given key.
// It allows multiple indexes to be associated with the same key.
func (m *MultiMap[T]) AddIndex(key string, indexName string, indexValue string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	indexKey := createIndexKey(indexName, indexValue)
	m.byIndex[indexKey] = key
	m.indexesByID[key] = append(m.indexesByID[key], indexKey)
}

// AddIndexByAnotherIndex adds an index to the map for the given index.
func (m *MultiMap[T]) AddIndexByAnotherIndex(
	indexName string, indexValue string,
	anotherIndexName string, anotherIndexValue string,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	indexKey := createIndexKey(indexName, indexValue)
	anotherIndexKey := createIndexKey(anotherIndexName, anotherIndexValue)
	m.byIndex[anotherIndexKey] = m.byIndex[indexKey]
	m.indexesByID[m.byIndex[indexKey]] = append(m.indexesByID[m.byIndex[indexKey]], anotherIndexKey)
}

// Load retrieves the value from the map with the given key.
// It returns the value and a boolean indicating whether the value was found.
func (m *MultiMap[T]) Load(key string) (T, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, ok := m.byID[key]

	return value, ok
}

// KeyByIndex retrieves the key from the map with the given index key.
func (m *MultiMap[T]) KeyByIndex(indexName string, indexValue string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.keyByIndex(createIndexKey(indexName, indexValue))
}

func (m *MultiMap[T]) keyByIndex(indexKey string) (string, bool) {
	key, ok := m.byIndex[indexKey]

	return key, ok
}

// LoadByIndex retrieves the value from the map with the given index key.
// It returns the value and a boolean indicating whether the value was found.
func (m *MultiMap[T]) LoadByIndex(indexName string, indexValue string) (T, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key, ok := m.byIndex[createIndexKey(indexName, indexValue)]
	if !ok {
		var zero T

		return zero, false
	}

	value, ok := m.byID[key]

	return value, ok
}

// Delete removes the value from the map with the given key.
func (m *MultiMap[T]) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.byID, key)

	if indexes, ok := m.indexesByID[key]; ok {
		for _, index := range indexes {
			delete(m.byIndex, index)
		}

		delete(m.indexesByID, key)
	}
}

// DeleteByIndex removes the value from the map with the given index key.
func (m *MultiMap[T]) DeleteByIndex(indexName string, indexValue string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	indexKey := createIndexKey(indexName, indexValue)
	if key, ok := m.byIndex[indexKey]; ok {
		delete(m.byID, key)
		delete(m.indexesByID, key)
		delete(m.byIndex, indexKey)
	}
}

// StoreOption is an interface for options that can be passed to the Store method.
type StoreOption interface {
	// markOption marks the option as an option.
	// It prevents to create StoreOption outside of package.
	markOption()
}

// WithIndexImpl is an implementation of StoreOption for index options.
type WithIndexImpl struct {
	indexKey string
}

func (w *WithIndexImpl) markOption() {}

// WithIndex creates a index option for the Store method.
func WithIndex(indexName, indexValue string) *WithIndexImpl {
	return &WithIndexImpl{
		indexKey: createIndexKey(indexName, indexValue),
	}
}

func createIndexKey(indexName string, indexValue string) string {
	return indexName + "/" + indexValue
}
