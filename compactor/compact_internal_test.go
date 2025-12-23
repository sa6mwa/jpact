package compactor

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"testing"
)

type singleByteReader struct {
	data []byte
	pos  int
}

func (r *singleByteReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	p[0] = r.data[r.pos]
	r.pos++
	return 1, nil
}

func TestNewCompactor(t *testing.T) {
	c := New()
	if c == nil {
		t.Fatalf("expected compactor")
	}

	var buf bytes.Buffer
	if err := c.CompactWriter(&buf, strings.NewReader(`{"a":1}`), 0); err != nil {
		t.Fatalf("compact failed: %v", err)
	}
	if got, want := buf.String(), `{"a":1}`; got != want {
		t.Fatalf("unexpected output: %q", got)
	}

	out, err := c.CompactToBuffer(strings.NewReader(`{"b":2}`), 0)
	if err != nil {
		t.Fatalf("compact buffer failed: %v", err)
	}
	if got, want := string(out), `{"b":2}`; got != want {
		t.Fatalf("unexpected buffer output: %q", got)
	}
}

func TestEmitASCIIBuffered(t *testing.T) {
	input := "abc\""
	var out bytes.Buffer
	c := &compactor{
		br: bufio.NewReader(strings.NewReader(input)),
		bw: bufio.NewWriter(&out),
	}

	if err := c.emitASCII(); err != nil {
		t.Fatalf("emitASCII failed: %v", err)
	}
	if err := c.bw.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	if got, want := out.String(), "abc"; got != want {
		t.Fatalf("unexpected output: %q", got)
	}

	next, err := c.br.ReadByte()
	if err != nil {
		t.Fatalf("expected next byte: %v", err)
	}
	if next != '"' {
		t.Fatalf("expected quote, got %q", next)
	}
}

func TestEmitASCIIUnbuffered(t *testing.T) {
	input := []byte("xyz")
	var out bytes.Buffer
	c := &compactor{
		br: bufio.NewReader(&singleByteReader{data: input}),
		bw: bufio.NewWriter(&out),
	}

	if err := c.emitASCII(); err != nil {
		t.Fatalf("emitASCII failed: %v", err)
	}
	if err := c.bw.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	if got, want := out.String(), "xyz"; got != want {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestUnreadByte(t *testing.T) {
	c := &compactor{br: bufio.NewReader(bytes.NewReader([]byte("a")))}
	b, err := c.readByte()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if b != 'a' {
		t.Fatalf("expected 'a', got %q", b)
	}
	c.unread()
	b, err = c.readByte()
	if err != nil {
		t.Fatalf("read failed after unread: %v", err)
	}
	if b != 'a' {
		t.Fatalf("expected 'a' after unread, got %q", b)
	}
}

func TestPoolResetsLargeBuffers(t *testing.T) {
	c := acquireCompactor(strings.NewReader(`{"a":1}`), io.Discard, 0)
	c.asciiBuf = make([]byte, 0, maxPooledBufCap+1)
	c.numBuf = make([]byte, 0, maxPooledBufCap+1)
	c.smallOut.Grow(maxPooledBufCap + 1)
	releaseCompactor(c)
}
