// Package jpact provides JSON compaction with selectable drivers.
//
// It exposes a small Compactor interface for streaming JSON compaction and
// convenience helpers that default to the internal driver.
//
// Usage:
//
//	c := jpact.New()
//	if err := c.CompactWriter(w, r, 0); err != nil {
//		// handle error
//	}
//
//	// Choose the jsonv2 driver.
//	c = jpact.New(jpact.WithDriver(jpact.DriverJSONv2))
//
// The maxBytes parameter limits the number of bytes read from the input
// stream; values <= 0 disable the limit.
//
// The package-level helpers (CompactWriter/CompactToBuffer) are always routed
// to the internal driver and do not mutate any package-level state.
package jpact
