package formatter_test

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/formatter"
)

type failFirstWriter struct {
	writes int
}

func (w *failFirstWriter) Write(data []byte) (int, error) {
	w.writes++
	if w.writes == 1 {
		return 0, io.ErrClosedPipe
	}

	return len(data), nil
}

func TestFormatTableWriteError(t *testing.T) {
	t.Parallel()

	for _, format := range []formatter.FormatType{formatter.SHORT, formatter.TEXT} {
		t.Run(string(format), func(t *testing.T) {
			t.Parallel()

			for _, name := range []string{"flush", "during\fwrite"} {
				t.Run(name, func(t *testing.T) {
					t.Parallel()

					writer := &failFirstWriter{}
					rows := []struct{ Name string }{{Name: name}}
					err := formatter.Format(writer, rows, format)
					require.ErrorIs(t, err, io.ErrClosedPipe)
					require.Equal(t, 1, writer.writes, "stop writing on the first error")
				})
			}
		})
	}
}
