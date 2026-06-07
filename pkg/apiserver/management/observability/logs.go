package observability

import (
	"log/slog"
	"os"
)

// UnsupportedLogFormatError is an error type that indicates an unsupported log format.
// It contains the unsupported log format.
type UnsupportedLogFormatError struct {
	LogFormat LogFormat
}

func newLogger(settings *Config) (*slog.Logger, error) {
	logWriter := os.Stdout
	logSettings := settings.Log

	options := &slog.HandlerOptions{
		AddSource:   true,
		Level:       logSettings.Level,
		ReplaceAttr: nil,
	}

	var handler slog.Handler

	switch logSettings.Format {
	case LogFormatJSON:
		handler = slog.NewJSONHandler(logWriter, options)
	case LogFormatText:
		handler = slog.NewTextHandler(logWriter, options)
	default:
		return nil, &UnsupportedLogFormatError{
			LogFormat: logSettings.Format,
		}
	}

	logger := slog.New(handler)

	return logger, nil
}

// Error implements the error interface for UnsupportedLogFormatError.
func (e *UnsupportedLogFormatError) Error() string {
	return "unsupported log format: " + string(e.LogFormat)
}
