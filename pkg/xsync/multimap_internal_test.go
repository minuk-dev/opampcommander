package xsync // To test the MultiMap's internal state, it's not xsync_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testcase[T any] struct {
	name       string
	operations func(m *MultiMap[T])
	expected   *MultiMap[T]
}

//nolint:funlen
func TestMultiMap_String(t *testing.T) {
	t.Parallel()

	tcs := []testcase[string]{
		{
			name:       "empty",
			operations: nil,
			expected: &MultiMap[string]{
				mu:          sync.RWMutex{},
				byID:        map[string]string{},
				byIndex:     map[string]string{},
				indexesByID: map[string][]string{},
			},
		},
		{
			name: "single add without index",
			operations: func(m *MultiMap[string]) {
				m.Store("key1", "value1")
			},
			expected: &MultiMap[string]{
				mu:          sync.RWMutex{},
				byID:        map[string]string{"key1": "value1"},
				byIndex:     map[string]string{},
				indexesByID: map[string][]string{},
			},
		},
		{
			name: "single add with index",
			operations: func(m *MultiMap[string]) {
				m.Store("key1", "value", WithIndex("index1", "value1"))
			},
			expected: &MultiMap[string]{
				mu:          sync.RWMutex{},
				byID:        map[string]string{"key1": "value"},
				byIndex:     map[string]string{"index1/value1": "key1"},
				indexesByID: map[string][]string{"key1": {"index1/value1"}},
			},
		},
		{
			name: "multiple adds with indexes",
			operations: func(m *MultiMap[string]) {
				m.Store("key1", "value", WithIndex("index1", "value1"), WithIndex("index2", "value2"))
			},
			expected: &MultiMap[string]{
				mu:          sync.RWMutex{},
				byID:        map[string]string{"key1": "value"},
				byIndex:     map[string]string{"index1/value1": "key1", "index2/value2": "key1"},
				indexesByID: map[string][]string{"key1": {"index1/value1", "index2/value2"}},
			},
		},
		{
			name: "delete by key",
			operations: func(m *MultiMap[string]) {
				m.Store("key1", "value", WithIndex("index1", "value1"))
				m.Store("key2", "value2", WithIndex("index2", "value2"))
				m.Delete("key1")
			},
			expected: &MultiMap[string]{
				mu:          sync.RWMutex{},
				byID:        map[string]string{"key2": "value2"},
				byIndex:     map[string]string{"index2/value2": "key2"},
				indexesByID: map[string][]string{"key2": {"index2/value2"}},
			},
		},
		{
			name: "delete by index",
			operations: func(m *MultiMap[string]) {
				m.Store("key1", "value", WithIndex("index1", "value1"))
				m.Store("key2", "value2", WithIndex("index2", "value2"))
				m.DeleteByIndex("index1", "value1")
			},
			expected: &MultiMap[string]{
				mu:          sync.RWMutex{},
				byID:        map[string]string{"key2": "value2"},
				byIndex:     map[string]string{"index2/value2": "key2"},
				indexesByID: map[string][]string{"key2": {"index2/value2"}},
			},
		},
		{
			name: "AddIndex",
			operations: func(m *MultiMap[string]) {
				m.Store("key1", "value")
				m.Store("key2", "value2")
				m.AddIndex("key1", "index1", "value1")
			},
			expected: &MultiMap[string]{
				mu:          sync.RWMutex{},
				byID:        map[string]string{"key1": "value", "key2": "value2"},
				byIndex:     map[string]string{"index1/value1": "key1"},
				indexesByID: map[string][]string{"key1": {"index1/value1"}},
			},
		},
	}

	for _, testcase := range tcs {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			multimap := NewMultiMap[string]()
			if testcase.operations != nil {
				testcase.operations(multimap)
			}

			assert.Equal(t, testcase.expected.byID, multimap.byID)
			assert.Equal(t, testcase.expected.byIndex, multimap.byIndex)
			assert.Equal(t, testcase.expected.indexesByID, multimap.indexesByID)
		})
	}
}
