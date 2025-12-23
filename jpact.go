package jpact

import (
	"io"

	"pkt.systems/jpact/compactor"
	"pkt.systems/jpact/jsonv2compactor"
)

// Compactor compacts JSON streams.
type Compactor interface {
	CompactWriter(w io.Writer, r io.Reader, maxBytes int64) error
	CompactToBuffer(r io.Reader, maxBytes int64) ([]byte, error)
}

// Driver selects the compaction implementation.
type Driver string

const (
	// DriverInternal selects the internal compactor.
	DriverInternal Driver = "internal"
	// DriverJSONv2 selects the jsonv2 compactor.
	DriverJSONv2 Driver = "jsonv2"
)

type options struct {
	driver Driver
}

// Option configures the jpact compactor.
type Option func(*options)

// WithDriver selects a compaction driver.
func WithDriver(driver Driver) Option {
	return func(o *options) {
		o.driver = driver
	}
}

// New returns a Compactor. The default driver is DriverInternal.
func New(opts ...Option) Compactor {
	cfg := options{driver: DriverInternal}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	switch cfg.driver {
	case DriverJSONv2:
		return jsonv2compactor.New()
	case DriverInternal, "":
		return compactor.New()
	default:
		return compactor.New()
	}
}

// CompactWriter compacts JSON using the default internal driver.
func CompactWriter(w io.Writer, r io.Reader, maxBytes int64) error {
	return compactor.CompactWriter(w, r, maxBytes)
}

// CompactToBuffer compacts JSON into memory using the default internal driver.
func CompactToBuffer(r io.Reader, maxBytes int64) ([]byte, error) {
	return compactor.CompactToBuffer(r, maxBytes)
}
