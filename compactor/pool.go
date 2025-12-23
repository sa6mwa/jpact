package compactor

import (
	"bufio"
	"bytes"
	"io"
	"sync"
)

const (
	defaultASCIIBufCap = 256
	defaultNumBufCap   = 64
	maxPooledBufCap    = 64 * 1024
)

var emptyReader = bytes.NewReader(nil)

var compactorPool = sync.Pool{
	New: func() any {
		return &compactor{
			br:       bufio.NewReader(emptyReader),
			bw:       bufio.NewWriter(io.Discard),
			asciiBuf: make([]byte, 0, defaultASCIIBufCap),
			numBuf:   make([]byte, 0, defaultNumBufCap),
		}
	},
}

func acquireCompactor(r io.Reader, w io.Writer, maxBytes int64) *compactor {
	c := compactorPool.Get().(*compactor)
	if c.br == nil {
		c.br = bufio.NewReader(r)
	} else {
		c.br.Reset(r)
	}
	if c.bw == nil {
		c.bw = bufio.NewWriter(w)
	} else {
		c.bw.Reset(w)
	}
	c.max = maxBytes
	c.read = 0
	c.stack = c.stackBuf[:0]
	c.topValueSeen = false
	c.prefix.Reset(nil)
	c.prefixReader.Reset(&c.prefix, nil)
	if c.smallOut.Cap() > maxPooledBufCap {
		c.smallOut = bytes.Buffer{}
	} else {
		c.smallOut.Reset()
	}

	if cap(c.asciiBuf) == 0 {
		c.asciiBuf = make([]byte, 0, defaultASCIIBufCap)
	} else if cap(c.asciiBuf) > maxPooledBufCap {
		c.asciiBuf = make([]byte, 0, defaultASCIIBufCap)
	} else {
		c.asciiBuf = c.asciiBuf[:0]
	}

	if cap(c.numBuf) == 0 {
		c.numBuf = make([]byte, 0, defaultNumBufCap)
	} else if cap(c.numBuf) > maxPooledBufCap {
		c.numBuf = make([]byte, 0, defaultNumBufCap)
	} else {
		c.numBuf = c.numBuf[:0]
	}

	return c
}

func releaseCompactor(c *compactor) {
	c.br.Reset(emptyReader)
	c.bw.Reset(io.Discard)
	c.stack = c.stackBuf[:0]
	c.topValueSeen = false
	c.max = 0
	c.read = 0
	c.prefix.Reset(nil)
	c.prefixReader.Reset(&c.prefix, nil)
	if c.smallOut.Cap() > maxPooledBufCap {
		c.smallOut = bytes.Buffer{}
	} else {
		c.smallOut.Reset()
	}

	if cap(c.asciiBuf) > maxPooledBufCap {
		c.asciiBuf = make([]byte, 0, defaultASCIIBufCap)
	} else {
		c.asciiBuf = c.asciiBuf[:0]
	}

	if cap(c.numBuf) > maxPooledBufCap {
		c.numBuf = make([]byte, 0, defaultNumBufCap)
	} else {
		c.numBuf = c.numBuf[:0]
	}

	compactorPool.Put(c)
}
