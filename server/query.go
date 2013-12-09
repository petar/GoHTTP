// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"io"
	"log"
	"strings"
	"time"
	"net/http"
	"net/http/httputil"
)

// Incoming requests are presented to the user as a Query object.
// Query allows users to respond to a request or to hijack the
// underlying ServerConn, which is typically needed for CONNECT
// requests.
type Query struct {
	Req *http.Request
	Ext map[string]interface{} // Extension-specific structures

	origPath string
	srv      *Server
	ssc      *StampedServerConn
	err      error
	fwd      bool // If true, the user has already called either Continue() or Hijack()
	hijacked bool

	t0       int64 // Time request was received
}

func newQueryErr(err error) *Query { return &Query{err: err} }

func (q *Query) getError() error { return q.err }

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
		panic("query zombie") // XXX: To be removed when issue 1563 fixed
	}
	go q.srv.read(q.ssc)
}

// Hijack() instructs the Server to stop managing the ServerConn
// that delivered the request underlying this Query. The connection is returned
// and the user becomes responsible for it.
// For every query returned by Server.Read(), the user must
// call either Continue() or Hijack(), but not both, and only once.
func (q *Query) Hijack() *httputil.ServerConn {
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
func (q *Query) Write(resp *http.Response) (err error) {
	if resp.Body != nil {
		defer func(b io.ReadCloser) { 
			b.Close() 
		}(resp.Body)
	}

	req := q.Req
	q.Req = nil
	ext := q.Ext
	q.Ext = nil

	// Invoke extensions in reverse order

	p := q.origPath
	revexts := q.srv.copyExtRev()
	for _, ec := range revexts {
		if strings.HasPrefix(p, ec.SubURL) {
			if err := ec.Ext.WriteResponse(resp, ext); err != nil {
				q.srv.bury(q.ssc)
				q.ssc = nil
				q.srv = nil
				return err
			}
		}
	}

	err = q.ssc.Write(req, resp)
	if err != nil {
		log.Printf("Response Write: %s\n", err)
		q.srv.bury(q.ssc)
		q.ssc = nil
		q.srv = nil
		return
	}
	q.srv.stats.AddReqRespTime(time.Now().UnixNano() - q.t0)
	q.srv.stats.IncResponse()
	return
}

func (q *Query) ContinueAndWrite(resp *http.Response) (err error) {
	q.Continue()
	return q.Write(resp)
}
