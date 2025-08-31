package port

import "github.com/minuk-dev/opampcommander/pkg/datastructure/sets"

type Indexer[T any] interface {
	Store[T]
	Index(indexName string, obj T) ([]any, error)
	IndexKeys(indexName, indexedValue string) ([]string, error)
	ListIndexFuncValues(indexName string) []string
	ByIndex(indexName, indexedValue string) ([]T, error)
	GetIndexers() Indexers[T]
	AddIndexers(newIndexers Indexers[T]) error
}

type Store[T any] interface {
	Add(obj T) error
	Update(obj T) error
	Delete(obj T) error
	List() []T
	ListKeys() []string
	Get(partialObj T) (obj *T, exists bool, err error)
	GetByKey(key string) (item *T, exists bool, err error)
	Replace([]T, string) error
}

type Indexers[T any] map[string]IndexFunc[T]

type IndexFunc[T any] func(obj T) ([]string, error)

// Index maps the indexed value to a set of keys in the store that match on that value
type Index map[string]sets.String

// Indices maps a name to an Index
type Indices map[string]Index
