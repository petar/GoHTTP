// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"io"
	"net"
	"os"
)

// runOnCloseReader wraps an io.ReadCloser, and executes a user-provided
// function run, after the first call to Close.
type runOnCloseReader struct {
	io.ReadCloser
	run func()
}

func NewRunOnCloseReader(c io.ReadCloser, f func()) *runOnCloseReader {
	return &runOnCloseReader{c, f}
}

func (t *runOnCloseReader) Close() os.Error {
	err := t.ReadCloser.Close()
	if t.run != nil {
		t.run()
		t.run = nil
	}
	return err
}

// runOnCloseConn wraps a net.Conn, and executes a user-provided
// function run, after the first call to Close.
type runOnCloseConn struct {
	net.Conn
	run func()
}

func NewRunOnCloseConn(c net.Conn, f func()) *runOnCloseConn {
	return &runOnCloseConn{c, f}
}

// XXX: make re-entrant
func (t *runOnCloseConn) Close() os.Error {
	err := t.Conn.Close()
	if t.run != nil {
		t.run()
		t.run = nil
	}
	return err
}
