// Package timeutil provides utility functions for time manipulation.
package timeutil

import "time"

//nolint:mnd,gosec
func UnixNanoToTime(nsec uint64) time.Time {
	sec := nsec / 1e9
	nsec %= 1e9

	return time.Unix(int64(sec), int64(nsec))
}
