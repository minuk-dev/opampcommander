// Package websocketutil provides pure functions for server-side WebSocket frames.
package websocketutil

import (
	"encoding/binary"
	"io"
)

// FrameSize returns the total size of an unmasked server frame.
// data must contain its complete header; the payload may be absent.
//
//nolint:mnd // RFC 6455 frame header lengths and length markers.
func FrameSize(data []byte) (uint64, error) {
	if len(data) < 2 {
		return 0, io.ErrShortBuffer
	}

	size := uint64(data[1] & 0x7f)
	switch size {
	case 126:
		if len(data) < 4 {
			return 0, io.ErrShortBuffer
		}

		return 4 + uint64(binary.BigEndian.Uint16(data[2:4])), nil
	case 127:
		if len(data) < 10 {
			return 0, io.ErrShortBuffer
		}

		return 10 + binary.BigEndian.Uint64(data[2:10]), nil
	default:
		return 2 + size, nil
	}
}

// GoingAwayFrame returns a FIN close frame with status 1001 (Going Away).
func GoingAwayFrame() []byte {
	return []byte("\x88\x02\x03\xe9")
}
