// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"net"
	"os/signal"
	"testing"
)

func sigh() {
	for {
		<-signal.Incoming
		panic()
	}
}

func serve(as *AsyncServer, t *testing.T) {
	q, err := as.Read()
	if err != nil {
		t.Fatalf("as, err: %s", err)
	}
	req := q.GetRequest()
	if req.Body != nil {
		req.Body.Close()
	}
	q.Continue()
	go serve(as, t)
	q.Write(newRespServiceUnavailable())
}

func TestAsyncServer(t *testing.T) {
	go sigh()
	l, err := net.Listen("tcp", ":4949")
	if err != nil {
		t.Fatalf("listen: %s", err)
	}
	as := NewAsyncServer(l, 10e9, 100)
	go serve(as, t)
	<-make(chan int) // wait forever
}
