package apiserver

import (
	"log/slog"
	"os"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

// UnsupportedLogFormatError is an error type that indicates an unsupported log format.
// It contains the unsupported log format.
type UnsupportedLogFormatError struct {
	LogFormat config.LogFormat
}

// NewLogger creates a new logger instance with default settings.
func NewLogger(settings *config.ServerSettings) (*slog.Logger, error) {
	logWriter := os.Stdout

	options := &slog.HandlerOptions{
		AddSource:   true,
		Level:       settings.LogLevel,
		ReplaceAttr: nil,
	}

	var handler slog.Handler

	switch settings.LogFormat {
	case config.LogFormatJSON:
		handler = slog.NewJSONHandler(logWriter, options)
	case config.LogFormatText:
		handler = slog.NewTextHandler(logWriter, options)
	default:
		return nil, &UnsupportedLogFormatError{
			LogFormat: settings.LogFormat,
		}
	}

	logger := slog.New(handler)

	logger.Debug("Logger initialized", slog.String("settings", settings.String()))

	return logger, nil
}

// Error implements the error interface for UnsupportedLogFormatError.
func (e *UnsupportedLogFormatError) Error() string {
	return "unsupported log format: " + string(e.LogFormat)
}
