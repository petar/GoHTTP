// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"sync"
)

var (
	htmlErrServiceUnavailable = "<html>" +
		"<head><title>503 Service Unavailable</title></head>\n" +
		"<body bgcolor=\"white\">\n" +
		"<center><h1>503 Service Unavailable</h1></center>\n" +
		"<hr><center>Go HTTP package</center>\n" +
		"</body></html>"
	respErrServiceUnavailable = &Response{
		Status:        "Service Unavailable",
		StatusCode:    503,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		RequestMethod: "GET",
		Body:          StringToBody(htmlErrServiceUnavailable),
		ContentLength: int64(len(htmlErrServiceUnavailable)),
		Close:         false,
	}
	htmlErrBadRequest = "<html>" +
		"<head><title>400 Bad Request</title></head>\n" +
		"<body bgcolor=\"white\">\n" +
		"<center><h1>400 Bad Request</h1></center>\n" +
		"<hr><center>Go HTTP package</center>\n" +
		"</body></html>"
	respErrBadRequest = &Response{
		Status:        "Bad Request",
		StatusCode:    400,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		RequestMethod: "GET",
		Body:          StringToBody(htmlErrBadRequest),
		ContentLength: int64(len(htmlErrBadRequest)),
		Close:         false,
	}
	respConnectionEstablished = &Response{
		Status:        "Connection Established",
		StatusCode:    200,
		Proto:         "HTTP/1.0",
		ProtoMajor:    1,
		ProtoMinor:    0,
		RequestMethod: "CONNECT",
		Close:         false,
		Header:        map[string]string{"Proxy-Agent": "Go-HTTP-package"},
	}
	// Lock on this while making copies of boilerplate responses
	blk sync.Mutex
)

func newRespServiceUnavailable() *Response {
	blk.Lock()
	defer blk.Unlock()
	r, err := DupResp(respErrServiceUnavailable)
	if err != nil {
		panic("boiler")
	}
	return r
}

func newRespBadRequest() *Response {
	blk.Lock()
	defer blk.Unlock()
	r, err := DupResp(respErrBadRequest)
	if err != nil {
		panic("boiler")
	}
	return r
}

func newRespConnectionEstablished() *Response {
	blk.Lock()
	defer blk.Unlock()
	r, err := DupResp(respConnectionEstablished)
	if err != nil {
		panic("boiler")
	}
	return r
}
