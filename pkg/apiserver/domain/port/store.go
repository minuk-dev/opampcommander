// Package domainport defines storage contracts shared by domain aggregates.
package domainport

import "context"

// Reader provides typed access to a stored resource. Implementations own the
// returned value's isolation and any read cache.
type Reader[K comparable, V any] interface {
	Get(ctx context.Context, key K) (V, error)
}

// Store provides typed resource reads, writes, and deletes. Values carry their
// identity; implementations extract the key and enforce their version and
// deletion rules. Put may update a value's resource version after acceptance.
type Store[K comparable, V any] interface {
	Reader[K, V]
	Put(ctx context.Context, value V) error
	Delete(ctx context.Context, key K) error
}
