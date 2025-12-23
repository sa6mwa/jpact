package jsonv2compactor

import (
	"testing"

	"pkt.systems/jpact/internal/compactortest"
)

func TestJSONv2Compactor(t *testing.T) {
	compactortest.Run(t, compactortest.Harness{
		Name:            "jsonv2",
		SmallThreshold:  smallJSONThreshold,
		CompactWriter:   CompactWriter,
		CompactToBuffer: CompactToBuffer,
	})
}

func FuzzJSONv2Compactor(f *testing.F) {
	compactortest.Fuzz(f, compactortest.Harness{
		Name:            "jsonv2",
		SmallThreshold:  smallJSONThreshold,
		CompactWriter:   CompactWriter,
		CompactToBuffer: CompactToBuffer,
	})
}
