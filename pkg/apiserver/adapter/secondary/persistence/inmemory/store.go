// Package inmemory provides in-memory implementations of the persistence ports.
//
// It backs "standalone" mode (config database type "inmemory"), letting the
// apiserver run as a single node without MongoDB. Data lives only in process
// memory and is lost on restart, so it is intended for development, demos, and
// small single-node deployments rather than production multi-server setups.
//
// Each repository wraps a generic [store], which reproduces the get/list/put
// semantics of the MongoDB common adapter: soft-delete-aware reads, stable
// insertion ordering, and cursor-based pagination via an opaque continue token.
package inmemory

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// item wraps a stored value with a stable, monotonically increasing insertion
// sequence. The sequence gives deterministic ordering and serves as the
// pagination cursor, mirroring MongoDB's ascending _id ordering.
type item[V any] struct {
	seq   uint64
	value V
}

// store is a concurrency-safe in-memory key/value collection that mirrors the
// list/get/put behaviour of the MongoDB common adapter, including soft-delete
// filtering and cursor pagination.
//
// V is normally a pointer to a domain model. The store deep-copies values on the
// way in (put) and on the way out (get/list/snapshot) via the clone function, so
// it never shares a mutable reference with its callers. This reproduces the
// MongoDB adapter's "fresh copy per call" semantics: a caller mutating a value it
// read does not affect the stored copy (only an explicit put persists), and two
// concurrent callers never race on the same object.
type store[K comparable, V any] struct {
	mu      sync.RWMutex
	items   map[K]*item[V]
	nextSeq uint64

	// clone deep-copies a value so the store shares no mutable state with callers.
	clone func(V) V

	// deletedAt extracts a value's soft-delete timestamp, or nil when the type
	// has no soft-delete concept. A non-nil, non-zero timestamp marks the value
	// as deleted and hides it from reads that do not opt into IncludeDeleted.
	deletedAt func(V) *time.Time
}

// newStore creates an empty store. clone must deep-copy a value (it is applied on
// every read and write to isolate the store from callers). Pass a deletedAt
// accessor for soft-deletable types, or nil for types that are hard-deleted
// (e.g. agents, servers).
func newStore[K comparable, V any](clone func(V) V, deletedAt func(V) *time.Time) *store[K, V] {
	return &store[K, V]{
		mu:        sync.RWMutex{},
		items:     make(map[K]*item[V]),
		nextSeq:   1,
		clone:     clone,
		deletedAt: deletedAt,
	}
}

func (s *store[K, V]) isDeleted(value V) bool {
	if s.deletedAt == nil {
		return false
	}

	deletedAt := s.deletedAt(value)

	return deletedAt != nil && !deletedAt.IsZero()
}

// get returns the value for key. It returns [port.ErrResourceNotExist] when the
// key is absent, or when the value is soft-deleted and options does not request
// IncludeDeleted.
func (s *store[K, V]) get(key K, options *model.GetOptions) (V, error) {
	var zero V

	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.items[key]
	if !ok {
		return zero, errResourceNotExist()
	}

	if s.isDeleted(entry.value) && (options == nil || !options.IncludeDeleted) {
		return zero, errResourceNotExist()
	}

	return s.clone(entry.value), nil
}

// put inserts or replaces the value for key, storing a deep copy so later caller
// mutations do not leak into the store. An existing key keeps its original
// insertion sequence so its position in list ordering is stable across updates.
func (s *store[K, V]) put(key K, value V) {
	stored := s.clone(value)

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.items[key]; ok {
		existing.value = stored

		return
	}

	s.items[key] = &item[V]{seq: s.nextSeq, value: stored}
	s.nextSeq++
}

// delete permanently removes key (hard delete). It returns
// [port.ErrResourceNotExist] when the key is absent.
func (s *store[K, V]) delete(key K) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[key]; !ok {
		return errResourceNotExist()
	}

	delete(s.items, key)

	return nil
}

// collect returns the entries matching the given criteria, ordered by insertion
// sequence. Entries are excluded when soft-deleted (unless includeDeleted), when
// their sequence is not strictly greater than afterSeq, or when filter rejects
// them. afterSeq of 0 disables the cursor.
//
// Returned entries hold deep copies of the stored values, cloned while the read
// lock is held, so callers never observe (or race on) the live stored objects.
func (s *store[K, V]) collect(includeDeleted bool, afterSeq uint64, filter func(V) bool) []item[V] {
	s.mu.RLock()
	entries := make([]item[V], 0, len(s.items))

	for _, entry := range s.items {
		switch {
		case !includeDeleted && s.isDeleted(entry.value):
			continue
		case entry.seq <= afterSeq:
			continue
		case filter != nil && !filter(entry.value):
			continue
		}

		entries = append(entries, item[V]{seq: entry.seq, value: s.clone(entry.value)})
	}

	s.mu.RUnlock()

	slices.SortFunc(entries, func(a, b item[V]) int {
		return cmp.Compare(a.seq, b.seq)
	})

	return entries
}

// snapshot returns the non-deleted values matching filter, in insertion order.
// Pass includeDeleted to also include soft-deleted values, and a nil filter to
// match everything. It is the building block for the typed list/find helpers
// the repositories need beyond plain pagination.
func (s *store[K, V]) snapshot(includeDeleted bool, filter func(V) bool) []V {
	entries := s.collect(includeDeleted, 0, filter)

	values := make([]V, 0, len(entries))
	for _, entry := range entries {
		values = append(values, entry.value)
	}

	return values
}

// list returns a paginated [model.ListResponse] for the values matching filter.
//
// It mirrors the MongoDB common adapter: soft-deleted values are excluded unless
// options.IncludeDeleted is set, results are ordered by insertion sequence, and
// the continue token resumes strictly after the last returned element. The
// returned Continue is the last element's cursor (empty when the page is empty),
// and RemainingItemCount is how many matching values follow this page.
func (s *store[K, V]) list(options *model.ListOptions, filter func(V) bool) (*model.ListResponse[V], error) {
	if options == nil {
		//exhaustruct:ignore
		options = &model.ListOptions{}
	}

	var afterSeq uint64

	if options.Continue != "" {
		parsed, err := strconv.ParseUint(options.Continue, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid continue token %q: %w", options.Continue, err)
		}

		afterSeq = parsed
	}

	candidates := s.collect(options.IncludeDeleted, afterSeq, filter)

	total := int64(len(candidates))

	page := candidates
	if options.Limit > 0 && int64(len(candidates)) > options.Limit {
		page = candidates[:options.Limit]
	}

	items := make([]V, 0, len(page))

	var lastSeq uint64

	for _, entry := range page {
		items = append(items, entry.value)
		lastSeq = entry.seq
	}

	continueToken := ""
	if len(page) > 0 {
		continueToken = strconv.FormatUint(lastSeq, 10)
	}

	return &model.ListResponse[V]{
		Items:              items,
		Continue:           continueToken,
		RemainingItemCount: total - int64(len(page)),
	}, nil
}
