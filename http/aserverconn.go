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

var ascAutoId int = 0

// An AsyncServerConn accepts requests and serves responses over an underlying
// HTTP connection, by acting as the server-side of the HTTP connection.
// It supports both keep-alive and pipelining.
// AsyncServerConn is not responsible for closing the underlying connection.
// The user must call Close to regain control of that connection and
// deal with it as desired.
type AsyncServerConn struct {
	id           int         // internal counter
	sc           *ServerConn // keepalive to client
	queries      list.List   // pipeline of queries
	rlk, wlk, lk sync.Mutex
}

type query struct {
	Req  *Request
	Resp *Response
}

// NewAsyncServerConn creates a new AsyncServerConn object over the connection c.
func NewAsyncServerConn(c net.Conn) *AsyncServerConn {
	asc := &AsyncServerConn{
		id: ascAutoId,
		sc: NewServerConn(c, nil),
	}
	ascAutoId++
	return asc
}

// Read() blocks until a new request is received. Read() is re-entrant.
func (asc *AsyncServerConn) Read() (req *Request, err os.Error) {

	asc.rlk.Lock()
	if asc.sc == nil {
		asc.rlk.Unlock()
		return nil, os.EBADF
	}
	req, err = asc.sc.Read()
	if err != nil {
		asc.rlk.Unlock()
		return
	}
	asc.lk.Lock()
	asc.queries.PushBack(query{req, nil})
	asc.lk.Unlock()

	if req.Body == nil {
		asc.rlk.Unlock()
	} else {
		req.Body = NewRunOnClose(req.Body, func() { asc.rlk.Unlock() })
	}

	return
}

// Write() answers the request req with resp.
// req must match a request returned from Read().
// Write() is re-entrant.
func (asc *AsyncServerConn) Write(req *Request, resp *Response) (err os.Error) {
	if req == nil || resp == nil {
		panic("asc, nil req/resp")
	}

	err = asc.placeResponse(req, resp)
	if err != nil {
		if resp.Body != nil {
			resp.Body.Close()
		}
		return
	}
	for {
		pq, err := asc.popServable()
		if err != nil {
			return
		}
		if pq == nil {
			return nil
		}

		asc.wlk.Lock()
		if asc.sc == nil {
			asc.wlk.Unlock()
			return os.EBADF
		}
		err = asc.sc.Write(pq.Resp)
		asc.wlk.Unlock()

		if err != nil {
			if pq.Resp.Body != nil {
				pq.Resp.Body.Close()
			}
			return err
		}
	}
	panic("unreachable")
}

// Close detaches the AsyncServerConn object from the underlying
// connection and closes the bodies of all unused responses passed to Serve.
// When done, it returns the underlying connection
// back to the user.
func (asc *AsyncServerConn) Close() (net.Conn, *bufio.Reader, os.Error) {
	asc.rlk.Lock()
	asc.wlk.Lock()
	asc.lk.Lock()

	if asc.sc == nil {
		asc.lk.Unlock()
		asc.wlk.Unlock()
		asc.rlk.Unlock()
		return nil, nil, os.EBADF
	}
	sc := asc.sc
	asc.sc = nil

	asc.lk.Unlock()
	asc.wlk.Unlock()
	asc.rlk.Unlock()

	c, r := sc.Close()

	// Must close the bodies of all unused responses
	asc.lk.Lock()
	elm := asc.queries.Front()
	for elm != nil {
		q := elm.Value.(query)
		if q.Resp != nil && q.Resp.Body != nil {
			q.Resp.Body.Close()
		}
		asc.queries.Remove(elm)
		elm = asc.queries.Front()
	}
	asc.lk.Unlock()

	return c, r, nil
}

func (asc *AsyncServerConn) placeResponse(req *Request, resp *Response) os.Error {
	asc.lk.Lock()
	defer asc.lk.Unlock()
	if asc.sc == nil {
		return os.EBADF
	}
	elm := asc.queries.Front()
	for elm != nil {
		q := elm.Value.(query)
		if q.Req == req {
			if q.Resp != nil {
				panic("asc, placing resp over existing one")
			}
			elm.Value = query{req, resp}
			return nil
		}
		elm = elm.Next()
	}
	panic("unreachable")
}

func (asc *AsyncServerConn) popServable() (*query, os.Error) {
	asc.lk.Lock()
	defer asc.lk.Unlock()
	if asc.sc == nil {
		return nil, os.EBADF
	}
	elm := asc.queries.Front()
	if elm == nil {
		return nil, nil
	}
	q := elm.Value.(query)
	if q.Resp == nil {
		return nil, nil
	}
	asc.queries.Remove(elm)
	return &q, nil
}
