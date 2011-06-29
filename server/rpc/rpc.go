// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"os"
	"rpc"
	"sync"
	"github.com/petar/GoHTTP/server"
)

// RPC is a Sub that acts as an HTTP RPC server.
// Requests are received in the form of HTTP GET requests with
// parameters in the URL, just like the ones produced by jQuery's
// AJAX calls. Responses are returned in the form of HTTP responses
// with return values in the form of a JSON object in the response
// body.
type RPC struct {
	rpcs       *rpc.Server // does not need locking, since re-entrant
	sync.Mutex             // protects auto
	auto       uint64
}

func NewRPC() *RPC {
	return &RPC{
		rpcs: rpc.NewServer(),
		auto: 1, // Start seq numbers from 1, so that 0 is always an invalid seq number
	}
}

func (rpcsub *RPC) Register(rcvr interface{}) os.Error {
	return rpcsub.rpcs.Register(rcvr)
}

func (rpcsub *RPC) RegisterName(name string, rcvr interface{}) os.Error {
	return rpcsub.rpcs.RegisterName(name, rcvr)
}

func (rpcsub *RPC) Serve(q *server.Query) {
	qx := &queryCodec{Query: q}
	rpcsub.Lock()
	qx.seq = rpcsub.auto
	rpcsub.auto++
	rpcsub.Unlock()
	q.Continue()
	rpcsub.rpcs.ServeCodec(qx)
}
