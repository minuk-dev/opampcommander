package clock

import (
	"time"

	k8sclock "k8s.io/utils/clock"
)

// RealClock implements Clock using the system clock.
type RealClock struct {
	k8sclock.RealClock
}

// NewRealClock returns a new RealClock using the system clock.
func NewRealClock() *RealClock {
	return &RealClock{RealClock: k8sclock.RealClock{}}
}

// Now returns the current local time.
func (c *RealClock) Now() time.Time {
	return c.RealClock.Now()
}

// Since returns the time elapsed since t.
func (c *RealClock) Since(t time.Time) time.Duration {
	return c.RealClock.Since(t)
}

// After waits for the duration to elapse and then sends the current time on the returned channel.
func (c *RealClock) After(d time.Duration) <-chan time.Time {
	return c.RealClock.After(d)
}

// NewTimer returns a new Timer that will send the current time on its channel after at least duration d.
//
//nolint:ireturn
func (c *RealClock) NewTimer(d time.Duration) k8sclock.Timer {
	return c.RealClock.NewTimer(d)
}

// Sleep pauses the current goroutine for at least the duration d.
func (c *RealClock) Sleep(d time.Duration) {
	c.RealClock.Sleep(d)
}

// Tick returns a channel that will send the time with a period specified by the duration argument.
func (c *RealClock) Tick(d time.Duration) <-chan time.Time {
	return c.RealClock.Tick(d)
}
