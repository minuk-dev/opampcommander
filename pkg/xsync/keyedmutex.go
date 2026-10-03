package xsync

import "sync"

// KeyedMutex serializes operations on the same key. Different keys use separate
// mutexes. Its zero value is ready to use; it must not be copied after first use.
type KeyedMutex[K comparable] struct {
	mu    sync.Mutex
	locks map[K]*keyedMutexEntry
}

type keyedMutexEntry struct {
	mu    sync.Mutex
	users int // Includes both the holder and waiters, keeping their mutex alive.
}

// Lock acquires the mutex for key.
func (m *KeyedMutex[K]) Lock(key K) {
	m.mu.Lock()

	if m.locks == nil {
		m.locks = make(map[K]*keyedMutexEntry)
	}

	entry := m.locks[key]
	if entry == nil {
		entry = &keyedMutexEntry{}
		m.locks[key] = entry
	}

	entry.users++
	m.mu.Unlock()
	entry.mu.Lock()
}

// Unlock releases the mutex for key and removes it when no holder or waiter
// remains. Unlock must be called only for a locked key.
func (m *KeyedMutex[K]) Unlock(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry := m.locks[key]
	entry.mu.Unlock()
	entry.users--

	if entry.users == 0 {
		delete(m.locks, key)
	}
}
