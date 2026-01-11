// Package clock provides a clock interface.
// clock package is a adapter class for k8s.io/utils/clock.
package clock

import (
	"time"

	k8sclock "k8s.io/utils/clock"
)

// Timer represents a timer.
type Timer interface {
	k8sclock.Timer
}

// Clock is from k8s.io/utils/clock.
type Clock interface {
	PassiveClock
	// After returns the channel of a new Timer.
	// This method does not allow to free/GC the backing timer before it fires. Use
	// NewTimer instead.
	After(d time.Duration) <-chan time.Time
	// NewTimer returns a new Timer.
	NewTimer(d time.Duration) Timer
	// Sleep sleeps for the provided duration d.
	// Consider making the sleep interruptible by using 'select' on a context channel and a timer channel.
	Sleep(d time.Duration)
	// Tick returns the channel of a new Ticker.
	// This method does not allow to free/GC the backing ticker. Use
	// NewTicker from WithTicker instead.
	Tick(d time.Duration) <-chan time.Time
}

// PassiveClock is a passive clock interface.
type PassiveClock interface {
	Now() time.Time
	Since(at time.Time) time.Duration
}
