//nolint:testpackage // Check that waiters retain the mutex and idle entries are removed.
package xsync

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyedMutexWaitersAndIndependentKeys(t *testing.T) {
	t.Parallel()

	var mutex KeyedMutex[string]

	mutex.Lock("agent")
	waiterEntered, releaseWaiter, waiterDone := make(chan struct{}), make(chan struct{}), make(chan struct{})

	go func() {
		mutex.Lock("agent")
		close(waiterEntered)
		<-releaseWaiter
		mutex.Unlock("agent")
		close(waiterDone)
	}()

	waiting := assert.Eventually(t, func() bool {
		mutex.mu.Lock()
		defer mutex.mu.Unlock()

		return mutex.locks["agent"].users == 2
	}, time.Second, time.Millisecond)
	otherDone := make(chan struct{})

	go func() {
		mutex.Lock("other")
		mutex.Unlock("other")
		close(otherDone)
	}()

	select {
	case <-otherDone:
	case <-time.After(time.Second):
		assert.Fail(t, "different keys must not block one another")
	}

	mutex.Unlock("agent")
	<-waiterEntered
	mutex.mu.Lock()
	assert.Equal(t, 1, mutex.locks["agent"].users, "the waiter's mutex must remain registered")
	mutex.mu.Unlock()
	close(releaseWaiter)
	<-waiterDone
	require.True(t, waiting)
	assert.Empty(t, mutex.locks)
}

func TestKeyedMutexConcurrentUse(t *testing.T) {
	t.Parallel()

	const workers, iterations = 8, 100

	var (
		mutex    KeyedMutex[int]
		counts   [2]int
		finished sync.WaitGroup
	)

	start := make(chan struct{})

	for worker := range workers {
		finished.Go(func() {
			<-start

			key := worker % len(counts)
			for range iterations {
				mutex.Lock(key)
				counts[key]++
				mutex.Unlock(key)
			}
		})
	}

	close(start)
	finished.Wait()
	assert.Equal(t, [2]int{workers * iterations / 2, workers * iterations / 2}, counts)
	assert.Empty(t, mutex.locks, "idle keys must not accumulate")
}
