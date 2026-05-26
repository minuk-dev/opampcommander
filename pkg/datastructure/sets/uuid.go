package sets

import (
	"slices"

	"github.com/google/uuid"
)

// UUID is a set of uuid.UUID values.
type UUID map[uuid.UUID]Empty

// NewUUID creates a new UUID set and inserts the given items.
func NewUUID(items ...uuid.UUID) UUID {
	ss := UUID{}
	ss.Insert(items...)

	return ss
}

// Insert adds items to the set.
func (s UUID) Insert(items ...uuid.UUID) {
	for _, item := range items {
		s[item] = Empty{}
	}
}

// Delete removes items from the set.
func (s UUID) Delete(items ...uuid.UUID) {
	for _, item := range items {
		delete(s, item)
	}
}

// Has checks if the set contains the item.
func (s UUID) Has(item uuid.UUID) bool {
	_, contained := s[item]

	return contained
}

// HasAll checks if the set contains all the items.
func (s UUID) HasAll(items ...uuid.UUID) bool {
	for _, item := range items {
		if !s.Has(item) {
			return false
		}
	}

	return true
}

// HasAny checks if the set contains any of the items.
func (s UUID) HasAny(items ...uuid.UUID) bool {
	return slices.ContainsFunc(items, s.Has)
}

// Len returns the number of items in the set.
func (s UUID) Len() int {
	return len(s)
}

// List returns the items in the set as a slice.
func (s UUID) List() []uuid.UUID {
	ret := make([]uuid.UUID, 0, len(s))
	for key := range s {
		ret = append(ret, key)
	}

	return ret
}

// Intersection returns a new set that is the intersection of the two sets.
func (s UUID) Intersection(other UUID) UUID {
	result := UUID{}
	// Iterate over the smaller set for efficiency.
	if len(s) < len(other) {
		for key := range s {
			if other.Has(key) {
				result.Insert(key)
			}
		}
	} else {
		for key := range other {
			if s.Has(key) {
				result.Insert(key)
			}
		}
	}

	return result
}
