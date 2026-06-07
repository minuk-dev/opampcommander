package opamp

import (
	"context"
	"log/slog"
)

// Logger is a struct which wraps the slog.Logger for supporting OpAMP logger interface.
type Logger struct {
	logger *slog.Logger
}

// Debugf is a method that logs a debug message.
func (l *Logger) Debugf(_ context.Context, format string, v ...any) {
	l.logger.Debug(format, v...)
}

// Errorf is a method that logs an error message.
func (l *Logger) Errorf(_ context.Context, format string, v ...any) {
	l.logger.Error(format, v...)
}
