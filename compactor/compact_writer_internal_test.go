package compactor

import (
	"bufio"
	"io"
	"strings"
	"testing"
)

func TestWriteLiteralUnexpectedEOF(t *testing.T) {
	c := &compactor{
		br: bufio.NewReader(strings.NewReader("ru")),
		bw: bufio.NewWriter(io.Discard),
	}
	if err := c.writeLiteral("true"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestWriteStringUnterminated(t *testing.T) {
	c := &compactor{
		br: bufio.NewReader(strings.NewReader("abc")),
		bw: bufio.NewWriter(io.Discard),
	}
	if err := c.writeString(); err == nil {
		t.Fatalf("expected error")
	}
}

func TestWriteNumberInvalidExponent(t *testing.T) {
	c := &compactor{
		br:     bufio.NewReader(strings.NewReader("e+")),
		bw:     bufio.NewWriter(io.Discard),
		numBuf: make([]byte, 0, defaultNumBufCap),
	}
	if err := c.writeNumber('1'); err == nil {
		t.Fatalf("expected error")
	}
}

func TestWriteStringInvalidEscape(t *testing.T) {
	c := &compactor{
		br: bufio.NewReader(strings.NewReader("\\q\"")),
		bw: bufio.NewWriter(io.Discard),
	}
	if err := c.writeString(); err == nil {
		t.Fatalf("expected error")
	}
}

func TestWriteStringInvalidUnicode(t *testing.T) {
	c := &compactor{
		br: bufio.NewReader(strings.NewReader("\\u12x4\"")),
		bw: bufio.NewWriter(io.Discard),
	}
	if err := c.writeString(); err == nil {
		t.Fatalf("expected error")
	}
}

func TestWriteStringUnterminatedEscape(t *testing.T) {
	c := &compactor{
		br: bufio.NewReader(strings.NewReader("\\")),
		bw: bufio.NewWriter(io.Discard),
	}
	if err := c.writeString(); err == nil {
		t.Fatalf("expected error")
	}
}

func TestWriteNumberLeadingZero(t *testing.T) {
	c := &compactor{
		br:     bufio.NewReader(strings.NewReader("1")),
		bw:     bufio.NewWriter(io.Discard),
		numBuf: make([]byte, 0, defaultNumBufCap),
	}
	if err := c.writeNumber('0'); err == nil {
		t.Fatalf("expected error")
	}
}

func TestWriteNumberFractionExponent(t *testing.T) {
	c := &compactor{
		br:     bufio.NewReader(strings.NewReader(".23e+4")),
		bw:     bufio.NewWriter(io.Discard),
		numBuf: make([]byte, 0, defaultNumBufCap),
	}
	if err := c.writeNumber('1'); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
