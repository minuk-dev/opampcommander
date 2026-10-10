package websocketutil_test

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/websocketutil"
)

func TestFrameSize(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		data []byte
		size uint64
	}{
		{"empty payload", []byte{0x82, 0}, 2},
		{"short length", []byte{0x82, 125}, 127},
		{"16-bit length", []byte{0x82, 126, 0x10, 0}, 4100},
		{"64-bit length", []byte{0x82, 127, 0, 0, 0, 0, 0, 1, 0x86, 0xa0}, 100010},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			original := append([]byte(nil), tc.data...)
			size, err := websocketutil.FrameSize(tc.data)
			require.NoError(t, err)
			assert.Equal(t, tc.size, size)
			assert.Equal(t, original, tc.data)
		})
	}
}

func TestFrameSize_RejectsShortHeaders(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"one byte", []byte{0x82}},
		{"short 16-bit length", []byte{0x82, 126, 0}},
		{"short 64-bit length", []byte{0x82, 127, 0, 0, 0, 0, 0, 0, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := websocketutil.FrameSize(tc.data)
			require.ErrorIs(t, err, io.ErrShortBuffer)
		})
	}
}

func TestGoingAwayFrame(t *testing.T) {
	t.Parallel()

	frame := websocketutil.GoingAwayFrame()
	assert.Equal(t, []byte{0x88, 0x02, 0x03, 0xe9}, frame)
	frame[0] = 0
	assert.Equal(t, []byte{0x88, 0x02, 0x03, 0xe9}, websocketutil.GoingAwayFrame())
}
