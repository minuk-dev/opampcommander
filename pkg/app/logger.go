package app

import (
	"log/slog"
	"os"
)

// LogFormat is a string type that represents the log format.
type LogFormat string

const (
	// LogFormatText represents the text log format.
	LogFormatText LogFormat = "text"
	// LogFormatJSON represents the JSON log format.
	LogFormatJSON LogFormat = "json"
)

// UnsupportedLogFormatError is an error type that indicates an unsupported log format.
// It contains the unsupported log format.
type UnsupportedLogFormatError struct {
	LogFormat LogFormat
}

// NewLogger creates a new logger instance with default settings.
func NewLogger(settings *ServerSettings) (*slog.Logger, error) {
	logWriter := os.Stdout

	options := &slog.HandlerOptions{
		AddSource:   true,
		Level:       settings.LogLevel,
		ReplaceAttr: nil,
	}

	var handler slog.Handler

	switch settings.LogFormat {
	case LogFormatJSON:
		handler = slog.NewJSONHandler(logWriter, options)
	case LogFormatText:
		handler = slog.NewTextHandler(logWriter, options)
	default:
		return nil, &UnsupportedLogFormatError{
			LogFormat: settings.LogFormat,
		}
	}

	logger := slog.New(handler)

	return logger, nil
}

// Error implements the error interface for UnsupportedLogFormatError.
func (e *UnsupportedLogFormatError) Error() string {
	return "unsupported log format: " + string(e.LogFormat)
}
