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

type stampedServerConn struct {
	*http.ServerConn
	stamp int64
	lk    sync.Mutex
}

func newStampedServerConn(c net.Conn, r *bufio.Reader) *stampedServerConn {
	return &stampedServerConn{
		ServerConn: http.NewServerConn(c, r),
		stamp:      time.Nanoseconds(),
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

func (ssc *stampedServerConn) Read() (req *http.Request, err os.Error) {
	ssc.touch()
	defer ssc.touch()
	return ssc.ServerConn.Read()
}

func (ssc *stampedServerConn) Write(req *http.Request, resp *http.Response) (err os.Error) {
	ssc.touch()
	defer ssc.touch()
	return ssc.ServerConn.Write(req, resp)
}
