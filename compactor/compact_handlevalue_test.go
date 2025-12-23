package compactor

import (
	"bufio"
	"io"
	"strings"
	"testing"
)

func newCompactorWithReader(input string) *compactor {
	return &compactor{
		br: bufio.NewReader(strings.NewReader(input)),
		bw: bufio.NewWriter(io.Discard),
	}
}

func TestHandleValueLiteralsAndNumbers(t *testing.T) {
	cases := []struct {
		name  string
		b     byte
		input string
	}{
		{name: "true", b: 't', input: "rue"},
		{name: "false", b: 'f', input: "alse"},
		{name: "null", b: 'n', input: "ull"},
		{name: "number", b: '1', input: "23"},
		{name: "string", b: '"', input: "hello\""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newCompactorWithReader(tc.input)
			if err := c.handleValue(tc.b); err != nil {
				t.Fatalf("handleValue failed: %v", err)
			}
		})
	}
}

func TestHandleValueContainers(t *testing.T) {
	c := newCompactorWithReader("")
	if err := c.handleValue('{'); err != nil {
		t.Fatalf("handleValue '{' failed: %v", err)
	}
	if err := c.handleValue('}'); err != nil {
		t.Fatalf("handleValue '}' failed: %v", err)
	}

	c = newCompactorWithReader("")
	if err := c.handleValue('['); err != nil {
		t.Fatalf("handleValue '[' failed: %v", err)
	}
	if err := c.handleValue(']'); err != nil {
		t.Fatalf("handleValue ']' failed: %v", err)
	}
}

func TestHandleValueErrors(t *testing.T) {
	cases := []struct {
		name  string
		b     byte
		stack []containerState
	}{
		{name: "unexpected_object_close", b: '}'},
		{name: "unexpected_array_close", b: ']'},
		{name: "unexpected_comma", b: ','},
		{name: "unexpected_colon", b: ':'},
		{name: "invalid_char", b: '@'},
		{name: "mismatched_object_close", b: '}', stack: []containerState{{typ: '['}}},
		{name: "mismatched_array_close", b: ']', stack: []containerState{{typ: '{'}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newCompactorWithReader("")
			if tc.stack != nil {
				c.stack = append(c.stack, tc.stack...)
			}
			if err := c.handleValue(tc.b); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}
