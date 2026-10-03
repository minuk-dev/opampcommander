package opamp

import (
	"sync"

	"github.com/google/uuid"
)

type agentConnectionLock struct {
	mu    sync.Mutex
	users int
}

// lockAgentConnection serializes certificate activation, message processing, and
// close cleanup for one agent on this server. In particular, replacement cannot
// occur between cleanup's active-connection check and its status/liveness writes.
// Entries include waiters and are removed when no operation uses them anymore.
func (s *Service) lockAgentConnection(uid uuid.UUID) func() {
	s.connectionLifecycleMu.Lock()

	if s.connectionLifecycleLocks == nil {
		s.connectionLifecycleLocks = make(map[uuid.UUID]*agentConnectionLock)
	}

	lock := s.connectionLifecycleLocks[uid]
	if lock == nil {
		lock = &agentConnectionLock{}
		s.connectionLifecycleLocks[uid] = lock
	}

	lock.users++
	s.connectionLifecycleMu.Unlock()
	lock.mu.Lock()

	return func() {
		lock.mu.Unlock()
		s.connectionLifecycleMu.Lock()
		defer s.connectionLifecycleMu.Unlock()

		lock.users--
		if lock.users == 0 {
			delete(s.connectionLifecycleLocks, uid)
		}
	}
}
