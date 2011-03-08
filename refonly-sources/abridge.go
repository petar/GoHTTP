// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"bufio"
	"io"
	"net"
	"os"
)

// Bridge forks two go-routines that forward traffic between (c1,r1) and (c2,r2).
// Bridge itself blocks until the EOF is reached, after which it closes both connections.
// A pair (ci,ri) consists of a connection ci together with any existing
// bufio.Reader ri for it, or nil if one does not exist.
func MakeBridge(c1 net.Conn, r1 *bufio.Reader, c2 net.Conn, r2 *bufio.Reader) (e1, e2 os.Error) {
	nf := make(chan int, 2)
	go bridgeOneWay(c1, r2, c2, nf)
	go bridgeOneWay(c2, r1, c1, nf)
	<-nf
	<-nf
	e1 = c1.Close()
	e2 = c2.Close()
	return
}

func bridgeOneWay(w io.Writer, r *bufio.Reader, cr net.Conn, ch chan<- int) {
	io.Copy(w, &leftoverReader{r, cr})
	ch <- 1
}

// leftoverReader takes a io.Reader and a bufio.Reader. First it
// reads from the bufio.Reader until it is drained, after which
// it continues reading from the io.Reader.
type leftoverReader struct {
	b *bufio.Reader
	r io.Reader
}

func (l *leftoverReader) Read(p []byte) (n int, err os.Error) {
	if l.b != nil {
		left := l.b.Buffered()
		n, err = l.b.Read(p[0:min(left, len(p))])
		if n == left {
			l.b = nil
		}
		return
	}
	return l.r.Read(p)
}

// ReqindReadCloser reads from an underlying io.ReadCloser while also
// recording everything read in a replay buffer.
type RewindReadCloser struct {
	raw    io.ReadCloser
	buf    []byte
	replay int // current pos in buf while replaying, or -1 if not replaying
	err    os.Error
}

func NewRewindReadCloser(rc io.ReadCloser, replaylim int) *RewindReadCloser {
	return &RewindReadCloser{
		raw:    rc,
		buf:    make([]byte, 0, replaylim),
		replay: -1,
	}
}

func (r *RewindReadCloser) Read(p []byte) (n int, err os.Error) {
	// We are replaying
	if r.replay >= 0 {
		n = copy(p, r.buf[r.replay:len(r.buf)])
		r.replay += n
		if r.replay == len(r.buf) {
			r.replay = -1
		}
		return
	}
	// Else, read from underlying
	if r.buf != nil {
		m := cap(r.buf) - len(r.buf)
		if m <= 0 {
			r.err = os.ERANGE // We cannot rewind any more
			r.buf = nil
			r.replay = -1
			return r.raw.Read(p)
		}
		m = min(len(p), m)
		n, err = r.raw.Read(p)
		if err == nil {
			l := len(r.buf)
			r.buf = r.buf[0 : l+n]
			copy(r.buf[l:l+n], p)
		} else if r.err != nil {
			r.err = err
			r.buf = nil
			r.replay = -1
		}
		return
	}
	return r.raw.Read(p)
}

func (r *RewindReadCloser) Close() os.Error {
	if r.err != nil {
		r.err = os.EBADF
		r.buf = nil
		r.replay = -1
	}
	return r.raw.Close()
}

// Rewind returns an error if Close() was already called, or
// if the amount read exceeded the replay buffer size.
func (r *RewindReadCloser) Rewind() os.Error {
	if r.err != nil {
		return r.err
	}
	if len(r.buf) > 0 {
		r.replay = 0
	}
	return nil
}

func min(x, y int) int {
	if x <= y {
		return x
	}
	return y
}
