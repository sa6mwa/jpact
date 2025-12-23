package compactortest

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"
)

const maxFuzzInput = 64 * 1024

// Harness provides compaction helpers for shared tests.
type Harness struct {
	Name            string
	SmallThreshold  int
	CompactWriter   func(w io.Writer, r io.Reader, maxBytes int64) error
	CompactToBuffer func(r io.Reader, maxBytes int64) ([]byte, error)
}

// Run executes the shared test suite for a compactor implementation.
func Run(t *testing.T, h Harness) {
	t.Helper()
	if h.CompactWriter == nil || h.CompactToBuffer == nil {
		t.Fatalf("compactortest: missing compactor functions")
	}

	t.Run("valid", func(t *testing.T) {
		for _, tc := range validCases() {
			t.Run(tc.name, func(t *testing.T) {
				assertParity(t, h, tc.input)
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		for _, tc := range invalidCases() {
			t.Run(tc.name, func(t *testing.T) {
				assertInvalid(t, h, tc.input)
			})
		}
	})

	t.Run("invalid_streaming", func(t *testing.T) {
		if h.SmallThreshold <= 0 {
			t.Skip("no threshold configured")
		}
		t.Run("trailing_comma_object", func(t *testing.T) {
			assertInvalid(t, h, largeObjectTrailingComma(h.SmallThreshold+64))
		})
		t.Run("trailing_comma_array", func(t *testing.T) {
			assertInvalid(t, h, largeArrayTrailingComma(h.SmallThreshold+64))
		})
		t.Run("invalid_number", func(t *testing.T) {
			assertInvalid(t, h, largeInvalidNumber(h.SmallThreshold+64))
		})
		t.Run("invalid_exponent", func(t *testing.T) {
			assertInvalid(t, h, largeInvalidExponent(h.SmallThreshold+64))
		})
		t.Run("invalid_unicode_escape", func(t *testing.T) {
			assertInvalid(t, h, largeInvalidUnicodeEscape(h.SmallThreshold+64))
		})
		t.Run("invalid_unicode_short", func(t *testing.T) {
			assertInvalid(t, h, largeInvalidUnicodeShort(h.SmallThreshold+64))
		})
		t.Run("invalid_escape_char", func(t *testing.T) {
			assertInvalid(t, h, largeInvalidEscapeChar(h.SmallThreshold+64))
		})
		t.Run("invalid_control_char", func(t *testing.T) {
			assertInvalid(t, h, largeInvalidControlChar(h.SmallThreshold+64))
		})
		t.Run("invalid_fraction", func(t *testing.T) {
			assertInvalid(t, h, largeInvalidFraction(h.SmallThreshold+64))
		})
		t.Run("invalid_exponent_sign", func(t *testing.T) {
			assertInvalid(t, h, largeInvalidExponentSign(h.SmallThreshold+64))
		})
		t.Run("invalid_number_minus", func(t *testing.T) {
			assertInvalid(t, h, largeInvalidNumberMinus(h.SmallThreshold+64))
		})
		t.Run("top_level_comma", func(t *testing.T) {
			assertInvalid(t, h, largeInvalidTopComma(h.SmallThreshold+64))
		})
		t.Run("top_level_colon", func(t *testing.T) {
			assertInvalid(t, h, largeInvalidTopColon(h.SmallThreshold+64))
		})
		t.Run("missing_object_comma", func(t *testing.T) {
			assertInvalid(t, h, largeMissingObjectComma(h.SmallThreshold+64))
		})
	})

	t.Run("maxbytes_small", func(t *testing.T) {
		input := []byte(`{"foo":"bar"}`)
		assertMaxBytes(t, h, input, 5)
	})

	t.Run("maxbytes_streaming", func(t *testing.T) {
		if h.SmallThreshold <= 0 {
			t.Skip("no threshold configured")
		}
		input := largeJSON(h.SmallThreshold + 128)
		assertMaxBytes(t, h, input, int64(h.SmallThreshold+16))
	})

	t.Run("reader_error_small", func(t *testing.T) {
		input := []byte(`{"foo":"bar"}`)
		assertReaderError(t, h, input, 2)
	})

	t.Run("reader_error_streaming", func(t *testing.T) {
		if h.SmallThreshold <= 0 {
			t.Skip("no threshold configured")
		}
		input := largeJSON(h.SmallThreshold + 64)
		assertReaderError(t, h, input, int64(h.SmallThreshold/2))
	})

	t.Run("writer_error_small", func(t *testing.T) {
		input := []byte(`{"foo":"bar"}`)
		assertWriterError(t, h, input, 1)
	})

	t.Run("writer_error_streaming", func(t *testing.T) {
		if h.SmallThreshold <= 0 {
			t.Skip("no threshold configured")
		}
		input := largeJSON(h.SmallThreshold + 64)
		assertWriterError(t, h, input, 1)
	})

	t.Run("valid_streaming", func(t *testing.T) {
		if h.SmallThreshold <= 0 {
			t.Skip("no threshold configured")
		}
		input := largeValidPayload(h.SmallThreshold + 256)
		assertParity(t, h, input)
	})
}

// Fuzz executes an aggressive fuzz test for a compactor implementation.
func Fuzz(f *testing.F, h Harness) {
	f.Helper()
	if h.CompactWriter == nil || h.CompactToBuffer == nil {
		f.Fatalf("compactortest: missing compactor functions")
	}

	for _, tc := range validCases() {
		f.Add(tc.input, uint16(0))
	}
	for _, tc := range invalidCases() {
		f.Add(tc.input, uint16(0))
	}
	if h.SmallThreshold > 0 {
		big := largeJSON(h.SmallThreshold + 256)
		f.Add(big, uint16(0))
		f.Add(big, uint16(h.SmallThreshold/2))
	}

	f.Fuzz(func(t *testing.T, input []byte, maxBytes uint16) {
		if len(input) > maxFuzzInput {
			input = input[:maxFuzzInput]
		}

		mb := int64(maxBytes)
		if maxBytes == 0 {
			mb = 0
		}

		ref, refErr := compactReference(input)

		var buf bytes.Buffer
		err := h.CompactWriter(&buf, bytes.NewReader(input), mb)
		out := buf.Bytes()

		outBuf, errBuf := h.CompactToBuffer(bytes.NewReader(input), mb)

		if (err == nil) != (errBuf == nil) {
			t.Fatalf("writer and buffer mismatch: writer=%v buffer=%v", err, errBuf)
		}
		if err == nil && !bytes.Equal(out, outBuf) {
			t.Fatalf("writer and buffer outputs differ")
		}

		if mb > 0 && int64(len(input)) > mb {
			if err == nil {
				t.Fatalf("expected maxBytes error but got none")
			}
			return
		}

		if refErr != nil {
			if err == nil {
				t.Fatalf("expected error for invalid JSON but got none")
			}
			return
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(out, ref) {
			t.Fatalf("output mismatch")
		}
	})
}

type testCase struct {
	name  string
	input []byte
}

func validCases() []testCase {
	cases := []string{
		` { "foo" : [ 1 , 2 , 3 ] } `,
		"\n\t{\"nested\": {\"a\": 1, \"b\":true}}",
		`{"empty": [   ] , "obj" : {   }}`,
		`{"string":"\"quoted\"","escape":"\\tab\n"}`,
		`{"unicode":"åäö"}`,
		` [ 0 , -1 , 3.1415 , 10e-3 ] `,
		`"plain string"`,
		`1234567890`,
		`true`,
		`false`,
		`null`,
		`{"n":-0.0,"exp":1E+9}`,
	}
	out := make([]testCase, 0, len(cases))
	for i, c := range cases {
		out = append(out, testCase{name: "case_" + strconv.Itoa(i), input: []byte(c)})
	}
	return out
}

func invalidCases() []testCase {
	cases := []testCase{
		{name: "empty", input: []byte("")},
		{name: "whitespace", input: []byte(" \n\t ")},
		{name: "unterminated_object", input: []byte(`{`)},
		{name: "unterminated_array", input: []byte(`[`)},
		{name: "missing_value", input: []byte(`{"a":}`)},
		{name: "missing_colon", input: []byte(`{"a"  "b"}`)},
		{name: "unexpected_comma_object", input: []byte(`{"a":1,,"b":2}`)},
		{name: "unexpected_comma_array", input: []byte(`[1,,2]`)},
		{name: "missing_array_comma", input: []byte(`[1 2]`)},
		{name: "leading_array_comma", input: []byte(`[,1]`)},
		{name: "missing_object_comma", input: []byte(`{"a":1 "b":2}`)},
		{name: "unquoted_object_key", input: []byte(`{1:2}`)},
		{name: "object_value_in_key", input: []byte(`{"a":1,2}`)},
		{name: "unexpected_colon", input: []byte(`:`)},
		{name: "unexpected_closing_array", input: []byte(`]`)},
		{name: "unexpected_closing_object", input: []byte(`}`)},
		{name: "trailing_comma_object", input: []byte(`{"a":1,}`)},
		{name: "trailing_comma_array", input: []byte(`[1,]`)},
		{name: "leading_zero", input: []byte(`{"a":00}`)},
		{name: "leading_zero2", input: []byte(`{"a":01}`)},
		{name: "bad_fraction", input: []byte(`{"a":1.}`)},
		{name: "bad_exponent", input: []byte(`{"a":1e}`)},
		{name: "bad_exponent_sign", input: []byte(`{"a":1e+}`)},
		{name: "invalid_literal", input: []byte(`{"a":tru}`)},
		{name: "invalid_escape", input: []byte(`{"a":"\x"}`)},
		{name: "invalid_unicode", input: []byte(`{"a":"\u12x4"}`)},
		{name: "invalid_unicode_short", input: []byte(`{"a":"\u12"}`)},
		{name: "invalid_number_minus", input: []byte(`{"a":-}`)},
		{name: "multiple_top_level", input: []byte(`0 1`)},
		{name: "multiple_top_level_objects", input: []byte(`{"a":1}{"b":2}`)},
		{name: "control_in_string", input: []byte("{\"a\":\"line1\nline2\"}")},
		{name: "invalid_utf8", input: []byte{0xff}},
	}
	return cases
}

func compactReference(input []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.Compact(&buf, input); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func assertParity(t *testing.T, h Harness, input []byte) {
	t.Helper()
	ref, err := compactReference(input)
	if err != nil {
		t.Fatalf("reference failed: %v", err)
	}

	var buf bytes.Buffer
	if err := h.CompactWriter(&buf, bytes.NewReader(input), 0); err != nil {
		t.Fatalf("compact writer failed: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), ref) {
		t.Fatalf("writer mismatch\n got: %q\nwant:%q", buf.String(), string(ref))
	}

	out, err := h.CompactToBuffer(bytes.NewReader(input), 0)
	if err != nil {
		t.Fatalf("compact buffer failed: %v", err)
	}
	if !bytes.Equal(out, ref) {
		t.Fatalf("buffer mismatch\n got: %q\nwant:%q", string(out), string(ref))
	}
}

func assertInvalid(t *testing.T, h Harness, input []byte) {
	t.Helper()
	if _, err := compactReference(input); err == nil {
		t.Fatalf("invalid case is accepted by encoding/json")
	}

	if err := h.CompactWriter(io.Discard, bytes.NewReader(input), 0); err == nil {
		t.Fatalf("expected writer error")
	}
	if _, err := h.CompactToBuffer(bytes.NewReader(input), 0); err == nil {
		t.Fatalf("expected buffer error")
	}
}

func assertMaxBytes(t *testing.T, h Harness, input []byte, maxBytes int64) {
	t.Helper()
	if maxBytes <= 0 {
		t.Fatalf("invalid maxBytes")
	}
	if int64(len(input)) <= maxBytes {
		t.Fatalf("input must exceed maxBytes")
	}

	if err := h.CompactWriter(io.Discard, bytes.NewReader(input), maxBytes); err == nil {
		t.Fatalf("expected maxBytes error")
	}
	if _, err := h.CompactToBuffer(bytes.NewReader(input), maxBytes); err == nil {
		t.Fatalf("expected maxBytes error (buffer)")
	}
}

func assertReaderError(t *testing.T, h Harness, input []byte, errAt int64) {
	t.Helper()
	if errAt <= 0 {
		t.Fatalf("invalid errAt")
	}
	if int64(len(input)) <= errAt {
		t.Fatalf("input must exceed errAt")
	}

	boom := errors.New("reader boom")
	reader := &errorReader{data: input, errAt: errAt, err: boom}

	if err := h.CompactWriter(io.Discard, reader, 0); err == nil || !errors.Is(err, boom) {
		t.Fatalf("expected reader error, got %v", err)
	}

	reader = &errorReader{data: input, errAt: errAt, err: boom}
	if _, err := h.CompactToBuffer(reader, 0); err == nil || !errors.Is(err, boom) {
		t.Fatalf("expected reader error (buffer), got %v", err)
	}
}

func assertWriterError(t *testing.T, h Harness, input []byte, limit int) {
	t.Helper()
	if limit <= 0 {
		t.Fatalf("invalid limit")
	}

	boom := errors.New("writer boom")
	writer := &errorWriter{limit: limit, err: boom}

	if err := h.CompactWriter(writer, bytes.NewReader(input), 0); err == nil || !errors.Is(err, boom) {
		t.Fatalf("expected writer error, got %v", err)
	}
}

func largeJSON(size int) []byte {
	payload := strings.Repeat("x", size)
	return []byte(`{"big":"` + payload + `"}`)
}

func largeValidPayload(size int) []byte {
	payload := strings.Repeat("x", size)
	return []byte(`{"big":"` + payload + `","n1":0,"n2":-12,"n3":3.14,"n4":1e+9,"n5":-2E-3,"t":true,"f":false,"nil":null,"esc":"\t\n\r\"","uni":"\u00e9","arr":[0,1,2],"obj":{"k":"v"}}`)
}

func largeInvalidNumber(size int) []byte {
	payload := strings.Repeat("x", size)
	return []byte(`{"big":"` + payload + `","n":01}`)
}

func largeInvalidExponent(size int) []byte {
	payload := strings.Repeat("x", size)
	return []byte(`{"big":"` + payload + `","n":1e}`)
}

func largeInvalidUnicodeEscape(size int) []byte {
	payload := strings.Repeat("x", size)
	return []byte(`{"big":"` + payload + `","bad":"\u12x4"}`)
}

func largeInvalidUnicodeShort(size int) []byte {
	payload := strings.Repeat("x", size)
	return []byte(`{"big":"` + payload + `","bad":"\u12"}`)
}

func largeInvalidEscapeChar(size int) []byte {
	payload := strings.Repeat("x", size)
	return []byte(`{"big":"` + payload + `","bad":"\q"}`)
}

func largeInvalidControlChar(size int) []byte {
	payload := strings.Repeat("x", size)
	out := append([]byte(`{"big":"`+payload+`","bad":"`), 0x1f)
	return append(out, []byte(`"}`)...)
}

func largeInvalidFraction(size int) []byte {
	payload := strings.Repeat("x", size)
	return []byte(`{"big":"` + payload + `","n":1.}`)
}

func largeInvalidExponentSign(size int) []byte {
	payload := strings.Repeat("x", size)
	return []byte(`{"big":"` + payload + `","n":1e+}`)
}

func largeInvalidNumberMinus(size int) []byte {
	payload := strings.Repeat("x", size)
	return []byte(`{"big":"` + payload + `","n":-}`)
}

func largeInvalidTopComma(size int) []byte {
	payload := strings.Repeat("x", size)
	return []byte(`,{"big":"` + payload + `"}`)
}

func largeInvalidTopColon(size int) []byte {
	payload := strings.Repeat("x", size)
	return []byte(`:{"big":"` + payload + `"}`)
}

func largeMissingObjectComma(size int) []byte {
	payload := strings.Repeat("x", size)
	return []byte(`{"big":"` + payload + `" "a":1}`)
}

func largeObjectTrailingComma(size int) []byte {
	payload := strings.Repeat("x", size)
	return []byte(`{"big":"` + payload + `",}`)
}

func largeArrayTrailingComma(size int) []byte {
	payload := strings.Repeat("x", size)
	return []byte(`["` + payload + `",]`)
}

type errorReader struct {
	data  []byte
	pos   int
	errAt int64
	err   error
}

func (r *errorReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	if int64(r.pos) >= r.errAt {
		return 0, r.err
	}

	max := int(r.errAt) - r.pos
	if max > len(p) {
		max = len(p)
	}
	if max > len(r.data)-r.pos {
		max = len(r.data) - r.pos
	}

	n := copy(p, r.data[r.pos:r.pos+max])
	r.pos += n
	if int64(r.pos) >= r.errAt {
		return n, r.err
	}
	return n, nil
}

type errorWriter struct {
	limit   int
	written int
	err     error
}

func (w *errorWriter) Write(p []byte) (int, error) {
	if w.written >= w.limit {
		return 0, w.err
	}

	remaining := w.limit - w.written
	if remaining <= 0 {
		return 0, w.err
	}
	if len(p) > remaining {
		w.written += remaining
		return remaining, w.err
	}
	w.written += len(p)
	return len(p), nil
}
