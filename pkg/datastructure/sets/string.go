package sets

// String is a set of strings.
type String map[string]Empty

// NewString creates a new String set and inserts the given items.
func NewString(items ...string) String {
	ss := String{}
	ss.Insert(items...)

	return ss
}

// StringKeySet creates a String set from the keys of the given map.
func StringKeySet[useless any](m map[string]useless) String {
	ret := String{}
	for key := range m {
		ret.Insert(key)
	}

	return ret
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

// List returns the items in the set as a slice.
func (s String) List() []string {
	ret := make([]string, 0, len(s))
	for key := range s {
		ret = append(ret, key)
	}

	return ret
}

// Intersection returns a new set that is the intersection of the two sets.
func (s String) Intersection(other String) String {
	result := String{}
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
