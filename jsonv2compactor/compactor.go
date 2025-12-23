package jsonv2compactor

import "io"

// Compactor compacts JSON streams.
type Compactor interface {
	CompactWriter(w io.Writer, r io.Reader, maxBytes int64) error
	CompactToBuffer(r io.Reader, maxBytes int64) ([]byte, error)
}

type driver struct{}

// New returns a Compactor using the jsonv2 implementation.
func New() Compactor {
	return driver{}
}

func (driver) CompactWriter(w io.Writer, r io.Reader, maxBytes int64) error {
	return CompactWriter(w, r, maxBytes)
}

func (driver) CompactToBuffer(r io.Reader, maxBytes int64) ([]byte, error) {
	return CompactToBuffer(r, maxBytes)
}
