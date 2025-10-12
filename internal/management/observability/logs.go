package observability

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

func newLogger(settings *config.ObservabilitySettings) (*slog.Logger, error) {
	logWriter := os.Stdout
	logSettings := settings.Log

	options := &slog.HandlerOptions{
		AddSource:   true,
		Level:       logSettings.Level,
		ReplaceAttr: nil,
	}

	var handler slog.Handler

	switch logSettings.Format {
	case config.LogFormatJSON:
		handler = slog.NewJSONHandler(logWriter, options)
	case config.LogFormatText:
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
