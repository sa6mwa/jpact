package jsonv2compactor

import (
	"bufio"
	"bytes"
	"io"
	"sync"

	"pkt.systems/jpact/jsonv2compactor/internal/jsonv2"
)

const maxPooledBufCap = 64 * 1024

var emptyReader = bytes.NewReader(nil)

var compactorPool = sync.Pool{
	New: func() any {
		return &compactor{
			tok: jsonv2.NewTokenizer(emptyReader, 0),
			bw:  bufio.NewWriter(io.Discard),
		}
	},
}

func acquireCompactor(r io.Reader, w io.Writer, maxBytes int64) *compactor {
	c := compactorPool.Get().(*compactor)
	if c.tok == nil {
		c.tok = jsonv2.NewTokenizer(r, maxBytes)
	} else {
		c.tok.Reset(r, maxBytes, maxPooledBufCap)
	}
	if c.bw == nil {
		c.bw = bufio.NewWriter(w)
	} else {
		c.bw.Reset(w)
	}
	c.stack = c.stackBuf[:0]
	c.topValueSeen = false
	c.prefix.Reset(nil)
	c.prefixReader.Reset(&c.prefix, nil)
	if c.smallOut.Cap() > maxPooledBufCap {
		c.smallOut = bytes.Buffer{}
	} else {
		c.smallOut.Reset()
	}
	return c
}

func releaseCompactor(c *compactor) {
	c.bw.Reset(io.Discard)
	if c.tok != nil {
		c.tok.Reset(nil, 0, maxPooledBufCap)
	}
	c.stack = c.stackBuf[:0]
	c.topValueSeen = false
	c.prefix.Reset(nil)
	c.prefixReader.Reset(&c.prefix, nil)
	if c.smallOut.Cap() > maxPooledBufCap {
		c.smallOut = bytes.Buffer{}
	} else {
		c.smallOut.Reset()
	}
	compactorPool.Put(c)
}
