package set

// String is a set of strings.
type String map[string]Empty

// NewString creates a new String set and inserts the given items.
func NewString(items ...string) String {
	ss := String{}
	ss.Insert(items...)
	return ss
}

// Insert adds items to the set.
func (s String) Insert(items ...string) {
	for _, item := range items {
		s[item] = Empty{}
	}
}

// Delete removes items from the set.
func (s String) Delete(items ...string) {
	for _, item := range items {
		delete(s, item)
	}
}

// Has checks if the set contains the item.
func (s String) Has(item string) bool {
	_, contained := s[item]
	return contained
}

// HasAll checks if the set contains all the items.
func (s String) HasAll(items ...string) bool {
	for _, item := range items {
		if !s.Has(item) {
			return false
		}
	}
	return true
}

// HasAny checks if the set contains any of the items.
func (s String) HasAny(items ...string) bool {
	for _, item := range items {
		if s.Has(item) {
			return true
		}
	}
	return false
}
