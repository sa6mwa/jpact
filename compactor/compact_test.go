package compactor

import (
	"testing"

	"pkt.systems/jpact/internal/compactortest"
)

func TestCompactor(t *testing.T) {
	compactortest.Run(t, compactortest.Harness{
		Name:            "internal",
		SmallThreshold:  smallJSONThreshold,
		CompactWriter:   CompactWriter,
		CompactToBuffer: CompactToBuffer,
	})
}

func FuzzCompactor(f *testing.F) {
	compactortest.Fuzz(f, compactortest.Harness{
		Name:            "internal",
		SmallThreshold:  smallJSONThreshold,
		CompactWriter:   CompactWriter,
		CompactToBuffer: CompactToBuffer,
	})
}
