// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This test demonstrates how to write a transparent HTTP/1.1
// proxy, using AsyncClient and AsyncServerConn.
// This example can be run with "gotest proxy_example.go". Stop
// the proxy with Ctrl-C. The user is invited to stress-test
// this proxy by holding the Ctrl-R button of the browser for
// extended periods.

package http

import (
	"fmt"
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

// Timeout after 10 idle second on any connection to server;
// Pipeline at most 2 request per connection;
// Attempt to fetch a request at most 2 times;
// Don't utilize more than 100 file descriptors;
var ac = NewAsyncClient(5e9, 2, 3, 100)

func serve(as *AsyncServer, t *testing.T) {
	q, err := as.Read()
	if err != nil {
		t.Fatalf("as, err: %s", err)
	}
	req := q.GetRequest()

	fmt.Printf("Request: %s\n", req.Host)
	go serve(as, t)

	if req.Method == "CONNECT" {
		resp, conn2 := ac.Connect(req)
		if conn2 == nil {
			q.Continue()
			q.Write(resp)
			return
		}
		q.Write(resp)
		asc := q.Hijack()
		conn1, r1, _ := asc.Close()
		if conn1 == nil {
			conn2.Close()
			return
		}
		MakeBridge(conn1, r1, conn2, nil)
		return
	}
	q.Continue()

	// Rewrite request
	req.Header["Proxy-Connection"] = "", false
	req.Header["Connection"] = "Keep-Alive"
	//req.Header["Keep-Alive"] = "30"
	url := req.URL
	req.URL = nil
	req.RawURL = url.RawPath
	if url.RawQuery != "" {
		req.RawURL += "?" + url.RawQuery
	}
	if url.Fragment != "" {
		req.RawURL += "#" + url.Fragment
	}

	// Dump request, use for debugging
	// dreq, _ := DumpRequest(req, false)
	// fmt.Printf("REQ:\n%s\n", string(dreq))

	resp := ac.Fetch(req)

	fmt.Printf("Response: %s\n", resp.Status)

	// Dump response, use for debugging
	// dresp, _ := DumpResponse(resp, false)
	// fmt.Printf("RESP:\n%s\n", string(dresp))

	resp.Close = false
	if resp.Header != nil {
		resp.Header["Connection"] = "", false
	}
	q.Write(resp)
}

func TestProxy(t *testing.T) {
	go sigh()
	l, err := net.Listen("tcp", ":4949")
	if err != nil {
		t.Fatalf("listen: %s", err)
	}
	as := NewAsyncServer(l, 20e9, 100)
	go serve(as, t)
	<-make(chan int)
}
