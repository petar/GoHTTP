// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"os"
	"path"
	"github.com/petar/GoHTTP/http"
)

// Incoming requests are presented to the user as a Query object.
// Query allows users to respond to a request or to hijack the
// underlying ServerConn, which is typically needed for CONNECT
// requests.
type Query struct {
	srv      *Server
	ssc      *stampedServerConn
	req      *http.Request
	err      os.Error
	fwd      bool	// If true, the user has already called either Continue() or Hijack()
	hijacked bool
}

func newQueryErr(err os.Error) *Query {
	return &Query{nil, nil, nil, err, false, false}
}


func (q *Query) getError() os.Error { return q.err }

// GetRequest() returns the underlying request. The result
// is never nil.
func (q *Query) GetRequest() *http.Request { return q.req }

func (q *Query) GetPath() string { 
	return path.Clean(q.req.URL.Path)
}

// Continue() indicates to the Server that it can continue
// listening for incoming requests on the ServerConn that
// delivered the request underlying this Query object.
// For every query returned by Server.Read(), the user must
// call either Continue() or Hijack(), but not both, exactly once.
func (q *Query) Continue() {
	if q.fwd {
		panic("continue/hijack")
	}
	q.fwd = true
	if q.srv == nil {
		panic("query zombie")	// XXX: To be removed when issue 1563 fixed
	}
	go q.srv.read(q.ssc)
}

// Hijack() instructs the Server to stop managing the ServerConn
// that delivered the request underlying this Query. The connection is returned
// and the user becomes responsible for it.
// For every query returned by Server.Read(), the user must
// call either Continue() or Hijack(), but not both, and only once.
func (q *Query) Hijack() *http.ServerConn {
	if q.fwd {
		panic("continue and hijack")
	}
	q.fwd = true
	q.hijacked = true
	srv := q.srv
	q.srv = nil
	ssc := q.ssc
	q.ssc = nil
	srv.unregister(ssc)
	return ssc.ServerConn
}

// Write sends resp back on the connection that produced the request.
// Any non-nil error returned pertains to the ServerConn and not
// to the Server as a whole.
func (q *Query) Write(resp *http.Response) (err os.Error) {
	req := q.req
	q.req = nil
	err = q.ssc.Write(req, resp)
	if err != nil {
		q.srv.bury(q.ssc)
		q.ssc = nil
		q.srv = nil
		return
	}
	return
}

func (q *Query) WriteAndContinue(resp *http.Response) (err os.Error) {
	err = q.Write(resp)
	if err == nil {
		q.Continue()
	}
	return err
}
