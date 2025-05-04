package xsync

import "sync"

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

// Store stores the value in the map with the given key.
// It overwrites the existing value if the key already exists.
func (m *MultiMap[T]) Store(key string, value T, opts ...StoreOption) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.byID[key] = value

	for _, opt := range opts {
		switch opt := opt.(type) {
		case *withIndex:
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
//
//nolint:ireturn
func (m *MultiMap[T]) Load(key string) (T, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, ok := m.byID[key]

	return value, ok
}

// LoadByIndex retrieves the value from the map with the given index key.
// It returns the value and a boolean indicating whether the value was found.
//
//nolint:ireturn
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

type withIndex struct {
	indexKey string
}

func (w *withIndex) markOption() {}

// WithIndex creates a index option for the Store method.
//
//nolint:ireturn
func WithIndex(indexName, indexValue string) StoreOption {
	return &withIndex{indexKey: createIndexKey(indexName, indexValue)}
}

func createIndexKey(indexName string, indexValue string) string {
	return indexName + "/" + indexValue
}
