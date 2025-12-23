package jsonv2compactor

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"pkt.systems/jpact/jsonv2compactor/internal/jsonv2"
)

type containerState struct {
	typ            byte
	objPhase       objPhase
	objCount       int
	arrExpectValue bool
	arrNeedComma   bool
	arrCount       int
}

const smallJSONThreshold = 2048
const defaultStackDepth = 64

type objPhase int

const (
	objExpectKey objPhase = iota
	objExpectColon
	objExpectValue
	objExpectComma
)

// CompactWriter streams JSON from r to w, stripping insignificant whitespace.
// maxBytes limits the number of bytes read from r (<=0 disables the limit).
func CompactWriter(w io.Writer, r io.Reader, maxBytes int64) error {
	c := acquireCompactor(r, w, maxBytes)
	defer releaseCompactor(c)
	if handled, err := c.prepareReader(r, maxBytes); handled {
		return err
	}
	if err := c.run(); err != nil {
		return err
	}
	return c.bw.Flush()
}

// CompactToBuffer returns the compacted JSON payload in memory.
func CompactToBuffer(r io.Reader, maxBytes int64) ([]byte, error) {
	var buf bytes.Buffer
	if err := CompactWriter(&buf, r, maxBytes); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type compactor struct {
	tok          *jsonv2.Tokenizer
	bw           *bufio.Writer
	stack        []containerState
	stackBuf     [defaultStackDepth]containerState
	topValueSeen bool
	prefix       bytes.Reader
	prefixReader prefixReader
	smallBuf     [smallJSONThreshold + 1]byte
	smallOut     bytes.Buffer
}

func (c *compactor) run() error {
	for {
		kind, data, err := c.tok.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if len(c.stack) != 0 {
					return fmt.Errorf("json: unexpected end of input")
				}
				if !c.topValueSeen {
					return fmt.Errorf("json: empty input")
				}
				return nil
			}
			return err
		}

		switch kind {
		case jsonv2.TokenBeginObject:
			if err := c.ensureValueContext(); err != nil {
				return err
			}
			if err := c.writeBytes(data); err != nil {
				return err
			}
			c.pushObject()

		case jsonv2.TokenEndObject:
			frame := c.currentFrame()
			if frame == nil || frame.typ != '{' {
				return fmt.Errorf("json: unexpected '}'")
			}
			if frame.objPhase == objExpectKey && frame.objCount != 0 {
				return fmt.Errorf("json: expected object key")
			}
			if frame.objPhase == objExpectColon || frame.objPhase == objExpectValue {
				return fmt.Errorf("json: unexpected '}'")
			}
			if err := c.writeBytes(data); err != nil {
				return err
			}
			c.popFrame()
			if err := c.valueComplete(); err != nil {
				return err
			}

		case jsonv2.TokenBeginArray:
			if err := c.ensureValueContext(); err != nil {
				return err
			}
			if err := c.writeBytes(data); err != nil {
				return err
			}
			c.pushArray()

		case jsonv2.TokenEndArray:
			frame := c.currentFrame()
			if frame == nil || frame.typ != '[' {
				return fmt.Errorf("json: unexpected ']'")
			}
			if frame.arrExpectValue && frame.arrCount != 0 {
				return fmt.Errorf("json: expected array value")
			}
			if err := c.writeBytes(data); err != nil {
				return err
			}
			c.popFrame()
			if err := c.valueComplete(); err != nil {
				return err
			}

		case jsonv2.TokenString:
			frame := c.currentFrame()
			if frame != nil && frame.typ == '{' && frame.objPhase == objExpectKey {
				if err := c.writeBytes(data); err != nil {
					return err
				}
				frame.objPhase = objExpectColon
				continue
			}
			if err := c.ensureValueContext(); err != nil {
				return err
			}
			if err := c.writeBytes(data); err != nil {
				return err
			}
			if err := c.valueComplete(); err != nil {
				return err
			}

		case jsonv2.TokenLiteral, jsonv2.TokenNumber:
			if err := c.ensureValueContext(); err != nil {
				return err
			}
			if err := c.writeBytes(data); err != nil {
				return err
			}
			if err := c.valueComplete(); err != nil {
				return err
			}

		case jsonv2.TokenColon:
			frame := c.currentFrame()
			if frame == nil || frame.typ != '{' || frame.objPhase != objExpectColon {
				return fmt.Errorf("json: unexpected colon")
			}
			if err := c.writeBytes(data); err != nil {
				return err
			}
			frame.objPhase = objExpectValue

		case jsonv2.TokenComma:
			frame := c.currentFrame()
			if frame == nil {
				return fmt.Errorf("json: unexpected comma")
			}
			if frame.typ == '{' {
				if frame.objPhase != objExpectComma {
					return fmt.Errorf("json: unexpected comma")
				}
				if err := c.writeBytes(data); err != nil {
					return err
				}
				frame.objPhase = objExpectKey
			} else {
				if !frame.arrNeedComma {
					return fmt.Errorf("json: unexpected comma")
				}
				if err := c.writeBytes(data); err != nil {
					return err
				}
				frame.arrExpectValue = true
				frame.arrNeedComma = false
			}

		default:
			return fmt.Errorf("json: unsupported token")
		}
	}
}

func (c *compactor) ensureValueContext() error {
	if len(c.stack) == 0 {
		if c.topValueSeen {
			return fmt.Errorf("json: multiple top-level values")
		}
		return nil
	}
	frame := c.currentFrame()
	if frame.typ == '{' {
		if frame.objPhase != objExpectValue {
			return fmt.Errorf("json: expected value after object key")
		}
		return nil
	}
	if frame.arrExpectValue {
		return nil
	}
	return fmt.Errorf("json: expected ',' or ']' in array")
}

func (c *compactor) pushObject() {
	state := containerState{typ: '{', objPhase: objExpectKey}
	c.stack = append(c.stack, state)
}

func (c *compactor) pushArray() {
	state := containerState{typ: '[', arrExpectValue: true}
	c.stack = append(c.stack, state)
}

func (c *compactor) popFrame() {
	if len(c.stack) > 0 {
		c.stack = c.stack[:len(c.stack)-1]
	}
}

func (c *compactor) currentFrame() *containerState {
	if len(c.stack) == 0 {
		return nil
	}
	return &c.stack[len(c.stack)-1]
}

func (c *compactor) writeBytes(data []byte) error {
	_, err := c.bw.Write(data)
	return err
}

func (c *compactor) valueComplete() error {
	if len(c.stack) == 0 {
		if c.topValueSeen {
			return fmt.Errorf("json: multiple top-level values")
		}
		c.topValueSeen = true
		return nil
	}

	frame := c.currentFrame()
	if frame.typ == '{' {
		frame.objPhase = objExpectComma
		frame.objCount++
		return nil
	}

	frame.arrExpectValue = false
	frame.arrNeedComma = true
	frame.arrCount++
	return nil
}

func (c *compactor) prepareReader(r io.Reader, maxBytes int64) (bool, error) {
	threshold := smallJSONThreshold
	if maxBytes > 0 && maxBytes < int64(threshold) {
		threshold = int(maxBytes)
	}
	if threshold <= 0 {
		return false, nil
	}

	limit := threshold + 1
	buf := c.smallBuf[:limit]
	total := 0

	for total < limit {
		n, err := r.Read(buf[total:limit])
		total += n
		if maxBytes > 0 && int64(total) > maxBytes {
			return true, fmt.Errorf("json: payload exceeds %d bytes", maxBytes)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				if total <= threshold {
					if c.smallOut.Cap() > maxPooledBufCap {
						c.smallOut = bytes.Buffer{}
					} else {
						c.smallOut.Reset()
					}
					c.smallOut.Grow(total)
					if err := json.Compact(&c.smallOut, buf[:total]); err != nil {
						return true, err
					}
					if _, err := c.smallOut.WriteTo(c.bw); err != nil {
						return true, err
					}
					return true, c.bw.Flush()
				}
				break
			}
			return true, err
		}
	}

	c.prefix.Reset(buf[:total])
	c.prefixReader.Reset(&c.prefix, r)
	c.tok.Reset(&c.prefixReader, maxBytes, maxPooledBufCap)
	return false, nil
}

type prefixReader struct {
	prefix *bytes.Reader
	rest   io.Reader
	done   bool
}

func (r *prefixReader) Reset(prefix *bytes.Reader, rest io.Reader) {
	r.prefix = prefix
	r.rest = rest
	r.done = false
}

func (r *prefixReader) Read(p []byte) (int, error) {
	if !r.done && r.prefix != nil {
		if r.prefix.Len() > 0 {
			n, err := r.prefix.Read(p)
			if n > 0 || err != io.EOF {
				return n, err
			}
		}
		r.done = true
	}
	if r.rest == nil {
		return 0, io.EOF
	}
	return r.rest.Read(p)
}
