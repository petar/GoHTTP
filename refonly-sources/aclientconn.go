// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"bufio"
	"container/list"
	"net"
	"os"
	"sync"
)

var accAutoId int = 0

// An AsyncClientConn fetches responses to requests over an underlying
// HTTP connection, by acting as the client-side of the HTTP connection.
// It supports both keep-alive and pipelining.
// AsyncClientConn is not responsible for closing the underlying connection.
// The user must call Close to regain control of that connection and
// deal with it as desired.
// NOTE: AsyncClientConn does not close request bodies if a failure occurs
// while writing the requests. The reason is to allow the user to resend the
// request perhaps on another connection. Typically the user will wrap a
// request body in a RewindReadCloser before passing it to Fetch(), to keep
// the body's consistency in the event of partial reads before failure.
type AsyncClientConn struct {
	id           int         // internal counter
	cc           *ClientConn // keepalive to server
	fetches      list.List   // pipeline of fetch requests
	rlk, wlk, lk sync.Mutex  // mutex for reading/writing on connection
}

type fetch struct {
	req     *Request
	onFetch func(*Response, os.Error)
}

type fetchResult struct {
	resp *Response
	err  os.Error
}

// NewAsyncClientConn creates a new AsyncClientConn object over the connection c.
func NewAsyncClientConn(c net.Conn) *AsyncClientConn {
	acc := &AsyncClientConn{
		id: accAutoId,
		cc: NewClientConn(c, nil),
	}
	accAutoId++
	return acc
}

// Pending returns the number of requests in the pipeline that have not
// been responded to yet.
func (acc *AsyncClientConn) Pending() int {
	acc.lk.Lock()
	defer acc.lk.Unlock()
	return acc.fetches.Len()
}

// Fetch enqueues the request req on the HTTP pipeline, and blocks
// until a response is available.
func (acc *AsyncClientConn) Fetch(req *Request) (resp *Response, err os.Error) {
	if req == nil {
		panic("acc, fetch with req=nil")
	}

	acc.wlk.Lock()
	if acc.cc == nil {
		acc.wlk.Unlock()
		return nil, os.EBADF
	}
	err = acc.cc.Write(req)
	acc.wlk.Unlock()
	if err != nil {
		return nil, err
	}

	// Put request in pipeline
	rch := make(chan fetchResult, 1)
	acc.lk.Lock()
	if acc.cc == nil {
		acc.lk.Unlock()
		return nil, os.EBADF
	}
	acc.fetches.PushBack(fetch{req,
		func(resp *Response, err os.Error) { rch <- fetchResult{resp, err} }})
	acc.lk.Unlock()

	// This reads one response from the connection, and it may not be
	// ours. But there is one read() call for every request in the pipeline,
	// so we will get our response eventually.
	acc.read()

	// Wait for response from the read side
	result := <-rch
	close(rch)

	resp = result.resp
	err = result.err
	return
}

func (acc *AsyncClientConn) read() {
	acc.rlk.Lock()
	if acc.cc == nil {
		acc.rlk.Unlock()
		acc.popFetch().onFetch(nil, os.EBADF)
		return
	}
	resp, err := acc.cc.Read()
	if resp != nil {
		if resp.Body == nil {
			acc.rlk.Unlock()
		} else {
			resp.Body = NewRunOnClose(resp.Body, func() { acc.rlk.Unlock() })
		}
		acc.popFetch().onFetch(resp, nil)
	} else {
		acc.rlk.Unlock()
		acc.popFetch().onFetch(nil, err)
	}
}

func (acc *AsyncClientConn) popFetch() fetch {
	acc.lk.Lock()
	elm := acc.fetches.Front()
	acc.fetches.Remove(elm)
	acc.lk.Unlock()
	return elm.Value.(fetch)
}

// Close detaches the AsyncClientConn object from the underlying
// connection. When done, it returns the underlying connection
// back to the user.
func (acc *AsyncClientConn) Close() (net.Conn, *bufio.Reader, os.Error) {
	acc.rlk.Lock()
	acc.wlk.Lock()
	acc.lk.Lock()

	if acc.cc == nil {
		acc.lk.Unlock()
		acc.wlk.Unlock()
		acc.rlk.Unlock()
		return nil, nil, os.EBADF
	}
	cc := acc.cc
	acc.cc = nil

	acc.lk.Unlock()
	acc.wlk.Unlock()
	acc.rlk.Unlock()

	c, r := cc.Close()

	return c, r, nil
}
