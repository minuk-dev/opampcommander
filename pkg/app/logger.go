package app

import "log/slog"

// NewLogger creates a new logger instance with default settings.
func NewLogger() *slog.Logger {
	logger := slog.Default()

	return logger
}
