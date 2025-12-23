package jpact

import (
	"bytes"
	"strings"
)

func Example() {
	input := `{"a": 1, "b": [true, false, null]}`
	var out bytes.Buffer

	compactor := New()
	_ = compactor.CompactWriter(&out, strings.NewReader(input), 0)
}

func Example_withDriver() {
	input := `{"a": 1}`
	var out bytes.Buffer

	compactor := New(WithDriver(DriverJSONv2))
	_ = compactor.CompactWriter(&out, strings.NewReader(input), 0)
}

func Example_packageHelpers() {
	input := `{"a": 1}`
	var out bytes.Buffer

	_ = CompactWriter(&out, strings.NewReader(input), 0)
}
