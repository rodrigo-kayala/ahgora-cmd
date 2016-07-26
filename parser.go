package main

import (
	"bufio"
	"bytes"
	"io"
)

// SkipTillReader struct
type SkipTillReader struct {
	rdr   *bufio.Reader
	delim []byte
	found bool
}

// NewSkipTillReader struct
func NewSkipTillReader(reader io.Reader, delim []byte) *SkipTillReader {
	return &SkipTillReader{
		rdr:   bufio.NewReader(reader),
		delim: delim,
		found: false,
	}
}

func (str *SkipTillReader) Read(p []byte) (n int, err error) {
	if str.found {
		return str.rdr.Read(p)
	}
	// search byte by byte for the delimiter
outer:
	for {
		for i := range str.delim {
			var c byte
			c, err = str.rdr.ReadByte()
			if err != nil {
				n = 0
				return
			}
			// doens't match so start over
			if str.delim[i] != c {
				continue outer
			}
		}
		str.found = true
		// we read the delimiter so add it back
		str.rdr = bufio.NewReader(io.MultiReader(bytes.NewReader(str.delim), str.rdr))
		return str.Read(p)
	}
}

// ReadTillReader struct
type ReadTillReader struct {
	rdr   *bufio.Reader
	delim []byte
	found bool
}

// NewReadTillReader struct
func NewReadTillReader(reader io.Reader, delim []byte) *ReadTillReader {
	return &ReadTillReader{
		rdr:   bufio.NewReader(reader),
		delim: delim,
		found: false,
	}
}

func (rtr *ReadTillReader) Read(p []byte) (n int, err error) {
	if rtr.found {
		return 0, io.EOF
	}
outer:
	for n < len(p) {
		for i := range rtr.delim {
			var c byte
			c, err = rtr.rdr.ReadByte()
			if err != nil && n > 0 {
				err = nil
				return
			} else if err != nil {
				return
			}
			p[n] = c
			n++
			if rtr.delim[i] != c {
				continue outer
			}
		}
		rtr.found = true
		break
	}
	if n == 0 {
		err = io.EOF
	}
	return
}
