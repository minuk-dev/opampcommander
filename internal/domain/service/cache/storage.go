package cache

import (
	"fmt"
	"sync"

	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/datastructure/sets"
)

type Storage[T any] interface {
	Add(key string, obj T)
	Update(key string, obj T)
	Delete(key string)
	Get(key string) (item *T, exists bool)
	List() []T
	ListKeys() []string
	Replace(map[string]T, string)
	Index(indexName string, obj T) ([]interface{}, error)
	IndexKeys(indexName, indexedValue string) ([]string, error)
	ListIndexFuncValues(name string) []string
	ByIndex(indexName, indexedValue string) ([]T, error)
	GetIndexers() port.Indexers[T]

	// AddIndexers adds more indexers to this store. This supports adding indexes after the store already has items.
	AddIndexers(newIndexers port.Indexers[T]) error
}

type ItemStore[T any] interface {
	Get(key string) *T
	Set(key string, obj T)
	Delete(key string)
	Values() []T
	Keys() []string
}

type IndexNotFoundError struct {
	IndexName string
}

func (e IndexNotFoundError) Error() string {
	return fmt.Sprintf("index %q not found", e.IndexName)
}

type IndexerConflictError struct {
	IndexNames []string
}

func (e IndexerConflictError) Error() string {
	return fmt.Sprintf("indexer conflict: %v", e.IndexNames)
}

type inMemoryItemStore[T any] struct {
	items map[string]T
}

func (s *inMemoryItemStore[T]) Get(key string) *T {
	item, exists := s.items[key]
	if !exists {
		return nil
	}

	return &item
}

func (s *inMemoryItemStore[T]) Set(key string, obj T) {
	s.items[key] = obj
}

func (s *inMemoryItemStore[T]) Delete(key string) {
	delete(s.items, key)
}

func (s *inMemoryItemStore[T]) Values() []T {
	values := make([]T, 0, len(s.items))
	for _, v := range s.items {
		values = append(values, v)
	}

	return values
}

func (s *inMemoryItemStore[T]) Keys() []string {
	keys := make([]string, 0, len(s.items))
	for k := range s.items {
		keys = append(keys, k)
	}

	return keys
}

func newInMemoryItemStore[T any](items map[string]T) *inMemoryItemStore[T] {
	return &inMemoryItemStore[T]{
		items: items,
	}
}

var _ Storage[any] = (*storage[any])(nil)

type storage[T any] struct {
	lock      sync.RWMutex
	itemStore ItemStore[T]
	index     *storeIndex[T]
}

func (s *storage[T]) Add(key string, obj T) {
	s.Update(key, obj)
}

func (s *storage[T]) Update(key string, obj T) {
	s.lock.Lock()
	defer s.lock.Unlock()

	oldObject := s.itemStore.Get(key)
	s.itemStore.Set(key, obj)
	s.index.updateIndices(oldObject, &obj, key)
}

func (s *storage[T]) Get(key string) (item *T, exists bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	obj := s.itemStore.Get(key)
	if obj == nil {
		return nil, false
	}

	return obj, true
}

func (s *storage[T]) Delete(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	oldObject := s.itemStore.Get(key)
	if oldObject != nil {
		s.index.updateIndices(oldObject, nil, key)
		s.itemStore.Delete(key)
	}
}

func (s *storage[T]) List() []T {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.itemStore.Values()
}

func (s *storage[T]) ListKeys() []string {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.itemStore.Keys()
}

func (s *storage[T]) Index(indexName string, obj T) ([]T, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	keys, err := s.index.getKeysFromIndex(indexName, obj)
	if err != nil {
		return nil, err
	}

	list := make([]T, 0, len(keys))
	for key := range keys {
		if item := s.itemStore.Get(key); item != nil {
			list = append(list, *item)
		}
	}

	return list, nil
}

func (s *storage[T]) ByIndex(indexName, indexedValue string) ([]T, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	keys, err := s.index.getKeysByIndex(indexName, indexedValue)
	if err != nil {
		return nil, err
	}

	list := make([]T, 0, len(keys))
	for key := range keys {
		if item := s.itemStore.Get(key); item != nil {
			list = append(list, *item)
		}
	}

	return list, nil
}

func (s *storage[T]) IndexKeys(indexName, indexedValue string) ([]string, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	keys, err := s.index.getKeysByIndex(indexName, indexedValue)
	if err != nil {
		return nil, err
	}

	return keys.List(), nil
}

func (s *storage[T]) ListIndexFuncValues(name string) []string {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.index.getIndexValues(name)
}

func (s *storage[T]) Replace(list map[string]T, resourceVersion string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.itemStore = newInMemoryItemStore(list)

	s.index.reset()

	for _, key := range s.itemStore.Keys() {
		s.index.updateIndices(nil, s.itemStore.Get(key), key)
	}
}

func (s *storage[T]) AddIndexers(newIndexers port.Indexers[T]) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	err := s.index.addIndexers(newIndexers)
	if err != nil {
		return err
	}

	// Reindex all existing items with the new indexers.
	for _, key := range s.itemStore.Keys() {
		for name := range newIndexers {
			s.index.updateSingleIndex(name, nil, s.itemStore.Get(key), key)
		}
	}

	return nil
}

func (s *storage[T]) GetIndexers() port.Indexers[T] {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.index.indexers
}

type storeIndex[T any] struct {
	indexers port.Indexers[T]
	indices  port.Indices
}

func (i *storeIndex[T]) getKeysFromIndex(indexName string, obj T) (sets.String, error) {
	indexFunc := i.indexers[indexName]
	if indexFunc == nil {
		return nil, IndexNotFoundError{indexName}
	}

	indexedValues, err := indexFunc(obj)
	if err != nil {
		return nil, fmt.Errorf("indexing: %w", err)
	}

	index := i.indices[indexName]

	var storeKeySet sets.String
	if len(indexedValues) == 1 {
		// In majority of cases, there is exactly one value matching.
		// Optimize the most common path - deduping is not needed here.
		storeKeySet = index[indexedValues[0]]
	} else {
		// Need to de-dupe the return list.
		// Since multiple keys are allowed, this can happen.
		storeKeySet = sets.String{}

		for _, indexedValue := range indexedValues {
			for key := range index[indexedValue] {
				storeKeySet.Insert(key)
			}
		}
	}

	return storeKeySet, nil
}

func (i *storeIndex[T]) getKeysByIndex(indexName, indexedValue string) (sets.String, error) {
	indexFunc := i.indexers[indexName]
	if indexFunc == nil {
		return nil, IndexNotFoundError{indexName}
	}

	index := i.indices[indexName]

	return index[indexedValue], nil
}

func (i *storeIndex[T]) getIndexValues(indexName string) []string {
	index := i.indices[indexName]

	names := make([]string, 0, len(index))
	for key := range index {
		names = append(names, key)
	}

	return names
}

func (i *storeIndex[T]) addIndexers(newIndexers port.Indexers[T]) error {
	oldKeys := sets.StringKeySet(i.indexers)
	newKeys := sets.StringKeySet(newIndexers)

	if oldKeys.HasAny(newKeys.List()...) {
		return IndexerConflictError{IndexNames: oldKeys.Intersection(newKeys).List()}
	}

	for k, v := range newIndexers {
		i.indexers[k] = v
	}

	return nil
}

// updateSingleIndex modifies the objects location in the named index:
// - for create you must provide only the newObj
// - for update you must provide both the oldObj and the newObj
// - for delete you must provide only the oldObj
// updateSingleIndex must be called from a function that already has a lock on the cache.
func (i *storeIndex[T]) updateSingleIndex(name string, oldObj *T, newObj *T, key string) {
	var oldIndexValues, indexValues []string

	indexFunc, ok := i.indexers[name]
	if !ok {
		// Should never happen. Caller is responsible for ensuring this exists, and should call with lock
		// held to avoid any races.
		panic(fmt.Errorf("indexer %q does not exist", name))
	}

	if oldObj != nil {
		var err error

		oldIndexValues, err = indexFunc(*oldObj)
		if err != nil {
			panic(fmt.Errorf("unable to calculate an index entry for key %q on index %q: %w", key, name, err))
		}
	} else {
		oldIndexValues = oldIndexValues[:0]
	}

	if newObj != nil {
		var err error

		indexValues, err = indexFunc(*newObj)
		if err != nil {
			panic(fmt.Errorf("unable to calculate an index entry for key %q on index %q: %w", key, name, err))
		}
	} else {
		indexValues = indexValues[:0]
	}

	index := i.indices[name]
	if index == nil {
		index = port.Index{}
		i.indices[name] = index
	}

	if len(indexValues) == 1 && len(oldIndexValues) == 1 && indexValues[0] == oldIndexValues[0] {
		// We optimize for the most common case where indexFunc returns a single value which has not been changed
		return
	}

	for _, value := range oldIndexValues {
		i.deleteKeyFromIndex(key, value, index)
	}

	for _, value := range indexValues {
		i.addKeyToIndex(key, value, index)
	}
}

// updateIndices modifies the objects location in the managed indexes:
// - for create you must provide only the newObj
// - for update you must provide both the oldObj and the newObj
// - for delete you must provide only the oldObj
// updateIndices must be called from a function that already has a lock on the cache.
func (i *storeIndex[T]) updateIndices(oldObj *T, newObj *T, key string) {
	for name := range i.indexers {
		i.updateSingleIndex(name, oldObj, newObj, key)
	}
}

func (i *storeIndex[T]) addKeyToIndex(key, indexValue string, index port.Index) {
	set := index[indexValue]
	if set == nil {
		set = sets.NewString()
		index[indexValue] = set
	}

	set.Insert(key)
}

func (i *storeIndex[T]) deleteKeyFromIndex(key, indexValue string, index port.Index) {
	set := index[indexValue]
	if set == nil {
		return
	}

	set.Delete(key)
	// If we don't delete the set when zero, indices with high cardinality
	// short lived resources can cause memory to increase over time from
	// unused empty sets. See `kubernetes/kubernetes/issues/84959`.
	if len(set) == 0 {
		delete(index, indexValue)
	}
}

func (i *storeIndex[T]) reset() {
	i.indices = port.Indices{}
}
