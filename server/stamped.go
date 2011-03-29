// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"bufio"
	"net"
	"os"
	"sync"
	"time"
	"github.com/petar/GoHTTP/http"
)

// StampedServerConn is an http.ServerConn which additionally
// keeps track of the last time the connection performed I/O.
type StampedServerConn struct {
	*http.ServerConn
	stamp int64
	lk    sync.Mutex
}

func NewStampedServerConn(c net.Conn, r *bufio.Reader) *StampedServerConn {
	return &StampedServerConn{
		ServerConn: http.NewServerConn(c, r),
		stamp:      time.Nanoseconds(),
	}
}

func (ssc *StampedServerConn) touch() {
	ssc.lk.Lock()
	defer ssc.lk.Unlock()
	ssc.stamp = time.Nanoseconds()
}

func (ssc *StampedServerConn) GetStamp() int64 {
	ssc.lk.Lock()
	defer ssc.lk.Unlock()
	return ssc.stamp
}

func (ssc *StampedServerConn) Read() (req *http.Request, err os.Error) {
	ssc.touch()
	defer ssc.touch()
	return ssc.ServerConn.Read()
}

func (ssc *StampedServerConn) Write(req *http.Request, resp *http.Response) (err os.Error) {
	ssc.touch()
	defer ssc.touch()
	return ssc.ServerConn.Write(req, resp)
}

// StampedClientConn is an http.ClientConn which additionally
// keeps track of the last time the connection performed I/O.
type StampedClientConn struct {
	*http.ClientConn
	stamp int64
	lk    sync.Mutex
}

func NewStampedClientConn(c net.Conn, r *bufio.Reader) *StampedClientConn {
	return &StampedClientConn{
		ClientConn: http.NewClientConn(c, r),
		stamp:      time.Nanoseconds(),
	}
}

func (scc *StampedClientConn) touch() {
	scc.lk.Lock()
	defer scc.lk.Unlock()
	scc.stamp = time.Nanoseconds()
}

func (scc *StampedClientConn) GetStamp() int64 {
	scc.lk.Lock()
	defer scc.lk.Unlock()
	return scc.stamp
}

func (scc *StampedClientConn) Read(req *http.Request) (resp *http.Response, err os.Error) {
	scc.touch()
	defer scc.touch()
	return scc.ClientConn.Read(req)
}

func (scc *StampedClientConn) Write(req *http.Request) (err os.Error) {
	scc.touch()
	defer scc.touch()
	return scc.ClientConn.Write(req)
}

// XXX: Should ClientConn.Do be wrapped as well?
