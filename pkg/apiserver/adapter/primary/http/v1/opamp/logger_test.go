//nolint:testpackage // white-box: Logger wraps an unexported slog.Logger field with no constructor.
package opamp

import (
	"log/slog"
	"testing"
)

// TestLogger exercises the OpAMP logger adapter methods. It must be white-box because the
// wrapped slog.Logger field is unexported and the type has no exported constructor.
func TestLogger(t *testing.T) {
	t.Parallel()

	logger := &Logger{logger: slog.Default()}

	logger.Debugf(t.Context(), "debug %s", "message")
	logger.Errorf(t.Context(), "error %s", "message")
}
