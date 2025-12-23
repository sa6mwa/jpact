package jpact

import (
	"bytes"
	"testing"
)

func TestDefaultCompactor(t *testing.T) {
	c := New()
	if c == nil {
		t.Fatalf("expected compactor")
	}
	input := []byte(`{"a": 1}`)
	var buf bytes.Buffer
	if err := c.CompactWriter(&buf, bytes.NewReader(input), 0); err != nil {
		t.Fatalf("compact failed: %v", err)
	}
	if got, want := buf.String(), `{"a":1}`; got != want {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestJSONv2Compactor(t *testing.T) {
	c := New(WithDriver(DriverJSONv2))
	if c == nil {
		t.Fatalf("expected compactor")
	}
	input := []byte(`{"a": 1}`)
	var buf bytes.Buffer
	if err := c.CompactWriter(&buf, bytes.NewReader(input), 0); err != nil {
		t.Fatalf("compact failed: %v", err)
	}
	if got, want := buf.String(), `{"a":1}`; got != want {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestPackageHelpers(t *testing.T) {
	input := []byte(`{"a": 1}`)
	var buf bytes.Buffer
	if err := CompactWriter(&buf, bytes.NewReader(input), 0); err != nil {
		t.Fatalf("compact failed: %v", err)
	}
	if got, want := buf.String(), `{"a":1}`; got != want {
		t.Fatalf("unexpected output: %q", got)
	}

	out, err := CompactToBuffer(bytes.NewReader(input), 0)
	if err != nil {
		t.Fatalf("compact buffer failed: %v", err)
	}
	if got, want := string(out), `{"a":1}`; got != want {
		t.Fatalf("unexpected output: %q", got)
	}
}
