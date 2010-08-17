// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"container/list"
	"net"
	"os"
	"sync"
	"time"
)

// AsyncServer automates the reception of incoming HTTP connections
// at a given net.Listener. AsyncServer accepts new connections and
// manages each one with an AsyncServerConn object. AsyncServer also
// makes sure that a pre-specified limit of active connections (i.e.
// file descriptors) is not exceeded.
type AsyncServer struct {
	tmo    int64 // keepalive timout
	listen net.Listener
	conns  map[*stampedServerConn]int
	qch    chan *Query
	fdl    FDLimiter
	lk     sync.Mutex
}

type stampedServerConn struct {
	*AsyncServerConn
	stamp int64
	lk    sync.Mutex
}

func newStampedServerConn(c net.Conn) *stampedServerConn {
	return &stampedServerConn{
		AsyncServerConn: NewAsyncServerConn(c),
		stamp:           time.Nanoseconds(),
	}
}

func (ssc *stampedServerConn) touch() {
	ssc.lk.Lock()
	defer ssc.lk.Unlock()
	ssc.stamp = time.Nanoseconds()
}

func (ssc *stampedServerConn) GetStamp() int64 {
	ssc.lk.Lock()
	defer ssc.lk.Unlock()
	return ssc.stamp
}

func (ssc *stampedServerConn) Read() (req *Request, err os.Error) {
	ssc.touch()
	defer ssc.touch()
	return ssc.AsyncServerConn.Read()
}

func (ssc *stampedServerConn) Write(req *Request, resp *Response) (err os.Error) {
	ssc.touch()
	defer ssc.touch()
	return ssc.AsyncServerConn.Write(req, resp)
}

// Incoming requests are presented to the user as a Query object.
// Query allows users to response to a request and to hijack the
// underlying AsyncServerConn, which is typically needed for CONNECT
// requests.
type Query struct {
	as            *AsyncServer
	ssc           *stampedServerConn
	req           *Request
	err           os.Error
	fwd, hijacked bool
}

// NewAsyncServer creates a new AsyncServer which listens for connections on l.
// New connections are automatically managed by AsyncServerConn objects with
// timout set to tmo nanoseconds. The AsyncServer object ensures that at no
// time more than fdlim file descriptors are allocated to incoming connections.
func NewAsyncServer(l net.Listener, tmo int64, fdlim int) *AsyncServer {
	if tmo < 2 {
		panic("as, timeout too small")
	}
	// TODO(petar): Perhaps a better design passes the FDLimiter as a parameter
	as := &AsyncServer{
		tmo:    tmo,
		listen: l,
		conns:  make(map[*stampedServerConn]int),
		qch:    make(chan *Query),
	}
	as.fdl.Init(fdlim)
	go as.acceptLoop()
	go as.expireLoop()
	return as
}

func (as *AsyncServer) GetFDLimiter() *FDLimiter { return &as.fdl }

func (as *AsyncServer) expireLoop() {
	for {
		as.lk.Lock()
		if as.listen == nil {
			as.lk.Unlock()
			return
		}
		now := time.Nanoseconds()
		kills := list.New()
		for ssc, _ := range as.conns {
			if now-ssc.GetStamp() >= as.tmo {
				kills.PushBack(ssc)
			}
		}
		as.lk.Unlock()
		elm := kills.Front()
		for elm != nil {
			ssc := elm.Value.(*stampedServerConn)
			as.bury(ssc)
			elm = elm.Next()
		}
		kills.Init()
		kills = nil
		time.Sleep(as.tmo)
	}
}

func (as *AsyncServer) acceptLoop() {
	for {
		as.lk.Lock()
		l := as.listen
		as.lk.Unlock()
		if l == nil {
			return
		}
		as.fdl.Lock()
		c, err := as.listen.Accept()
		if err != nil {
			if c != nil {
				c.Close()
			}
			as.fdl.Unlock()
			as.qch <- &Query{nil, nil, nil, err, false, false}
			return
		}
		c.(*net.TCPConn).SetKeepAlive(true)
		err = c.SetReadTimeout(as.tmo)
		if err != nil {
			c.Close()
			as.fdl.Unlock()
			as.qch <- &Query{nil, nil, nil, err, false, false}
			return
		}
		c = NewConnRunOnClose(c, func() { as.fdl.Unlock() })
		ssc := newStampedServerConn(c)
		ok := as.register(ssc)
		if !ok {
			ssc.Close()
			c.Close()
		}
		go as.read(ssc)
	}
}

// Read() waits until a new request is received. The request is
// returned in the form of a Query object. A returned error
// indicates that the AsyncServer cannot accept new connections,
// and the user us expected to call Shutdown(), perhaps after serving
// outstanding queries.
func (as *AsyncServer) Read() (query *Query, err os.Error) {
	q := <-as.qch
	as.lk.Lock()
	if closed(as.qch) {
		as.lk.Unlock()
		return nil, os.EBADF
	}
	as.lk.Unlock()
	if err = q.getError(); err != nil {
		return nil, err
	}
	return q, nil
}

func (q *Query) getError() os.Error { return q.err }

// GetRequest() returns the underlying request. The result
// is never nil.
func (q *Query) GetRequest() *Request { return q.req }

// Continue() indicates to the AsyncServer that it can continue
// listening for incoming requests on the AsyncServerConn that
// delivered the request underlying this Query object.
// For every query returned by AsyncServer.Read(), the user must
// call either Continue() or Hijack(), but not both, and only once.
func (q *Query) Continue() {
	if q.fwd {
		panic("as, query, continue/hijack")
	}
	q.fwd = true
	go q.as.read(q.ssc)
}

// Hijack() instructs the AsyncServer to stop managing the AsyncServerConn
// that delivered the request underlying this Query. The connection is returned
// and the user becomes responsible for it.
// For every query returned by AsyncServer.Read(), the user must
// call either Continue() or Hijack(), but not both, and only once.
func (q *Query) Hijack() *AsyncServerConn {
	if q.fwd {
		panic("as, query, continue/hijack")
	}
	q.fwd = true
	q.hijacked = true
	as := q.as
	q.as = nil
	ssc := q.ssc
	q.ssc = nil
	as.unregister(ssc)
	return ssc.AsyncServerConn
}

// Write sends resp back on the connection that produced the request.
// Any non-nil error returned pertains to the AsyncServerConn and not
// to the AsyncServer as a whole.
func (q *Query) Write(resp *Response) (err os.Error) {
	req := q.req
	q.req = nil
	err = q.ssc.Write(req, resp)
	if err != nil {
		q.as.bury(q.ssc)
		q.ssc = nil
		q.as = nil
		return
	}
	return
}

func (as *AsyncServer) read(ssc *stampedServerConn) {
	for {
		req, err := ssc.Read()
		perr, ok := err.(*os.PathError)
		if ok && perr.Error == os.EAGAIN {
			as.bury(ssc)
			return
		}
		if err != nil {
			// TODO(petar): Technically, a read side error should not terminate
			// the ASC, if there are outstanding requests to be answered,
			// since the write side might still be healthy. But this is
			// virtually never the case with TCP, so we currently go for simplicity
			// and just close the connection.
			as.bury(ssc)
			return
		}
		as.qch <- &Query{as, ssc, req, nil, false, false}
		return
	}
}

func (as *AsyncServer) register(ssc *stampedServerConn) bool {
	as.lk.Lock()
	defer as.lk.Unlock()
	if closed(as.qch) {
		return false
	}
	if _, present := as.conns[ssc]; present {
		panic("as, register twice")
	}
	as.conns[ssc] = 1
	return true
}

func (as *AsyncServer) unregister(ssc *stampedServerConn) {
	as.lk.Lock()
	defer as.lk.Unlock()
	as.conns[ssc] = 0, false
}

func (as *AsyncServer) bury(ssc *stampedServerConn) {
	as.unregister(ssc)
	c, _, _ := ssc.Close()
	if c != nil {
		c.Close()
	}
}

// Shutdown closes the AsyncServer by closing the underlying
// net.Listener object. The user should not use any AsyncServer
// or Query methods after a call to Shutdown.
func (as *AsyncServer) Shutdown() (err os.Error) {
	// First, close the listener
	as.lk.Lock()
	var l net.Listener
	l, as.listen = as.listen, nil
	close(as.qch)
	as.lk.Unlock()
	if l != nil {
		err = l.Close()
	}
	// Then, force-close all open connections
	as.lk.Lock()
	for ssc, _ := range as.conns {
		c, _, _ := ssc.Close()
		if c != nil {
			c.Close()
		}
		as.conns[ssc] = 0, false
	}
	as.lk.Unlock()
	return
}
