package config

import (
	"errors"
	"time"
)

const (
	defaultDrainWindow     = 10 * time.Second
	defaultShutdownTimeout = 30 * time.Second
)

// ErrInvalidShutdown indicates a shutdown window that cannot fit within its timeout.
var ErrInvalidShutdown = errors.New("shutdown requires 0 <= drainWindow < timeout")

// ShutdownSettings bounds graceful shutdown, including the staggered WebSocket drain.
type ShutdownSettings struct {
	DrainWindow time.Duration `mapstructure:"drainWindow"`
	Timeout     time.Duration `mapstructure:"timeout"`
}

// WithDefaults supplies the default drain window and total shutdown timeout.
func (s ShutdownSettings) WithDefaults() ShutdownSettings {
	if s.DrainWindow == 0 {
		s.DrainWindow = defaultDrainWindow
	}

	if s.Timeout == 0 {
		s.Timeout = defaultShutdownTimeout
	}

	return s
}

// Validate leaves time after the drain window for close handshakes and HTTP shutdown.
func (s ShutdownSettings) Validate() error {
	s = s.WithDefaults()
	if s.DrainWindow < 0 || s.Timeout <= s.DrainWindow {
		return ErrInvalidShutdown
	}

	return nil
}
