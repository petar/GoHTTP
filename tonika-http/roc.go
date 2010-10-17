// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"io"
	"net"
	"os"
)

// runOnClose wraps an io.Closer, and executes a user-provided
// function run, after the first call to Close.
type runOnClose struct {
	io.ReadCloser
	run func()
}

func NewRunOnClose(c io.ReadCloser, f func()) *runOnClose {
	return &runOnClose{c, f}
}

func (roc *runOnClose) Close() os.Error {
	err := roc.ReadCloser.Close()
	if roc.run != nil {
		roc.run()
		roc.run = nil
	}
	return err
}

// connRunOnClose wraps a net.Conn, and executes a user-provided
// function run, after the first call to Close.
type connRunOnClose struct {
	net.Conn
	run func()
}

func NewConnRunOnClose(c net.Conn, f func()) *connRunOnClose {
	return &connRunOnClose{c, f}
}

func (croc *connRunOnClose) Close() os.Error {
	err := croc.Conn.Close()
	if croc.run != nil {
		croc.run()
		croc.run = nil
	}
	return err
}
