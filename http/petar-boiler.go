// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

func NewResponse200(req *Request) *Response {
	return &Response{
		Status:        "OK",
		StatusCode:    200,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Request:       req,
		Close:         false,
	}
}

func NewResponse200Bytes(req *Request, b []byte) *Response {
	if len(b) == 0 {
		return NewResponse200(req)
	}
	return &Response{
		Status:        "OK",
		StatusCode:    200,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Request:       req,
		Body:          NewBodyBytes(b),
		ContentLength: int64(len(b)),
		Close:         false,
	}
}

func NewResponse200CONNECT(req *Request) *Response {
	return &Response{
		Status:        "Connection Established",
		StatusCode:    200,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Request:       req,
		Close:         false,
		Header:        Header{"Proxy-Agent": []string{"Go-HTTP-package"}},
	}
}

func NewResponse500(req *Request) *Response {
	html := "<html>" +
		"<head><title>500 Internal Server Error</title></head>\n" +
		"<body bgcolor=\"white\">\n" +
		"<center><h1>500 Internal Server Error</h1></center>\n" +
		"<hr><center>Go HTTP package</center>\n" +
		"</body></html>"
	return &Response{
		Status:        "Internal Server Error",
		StatusCode:    500,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Request:       req,
		Body:          NewBodyString(html),
		ContentLength: int64(len(html)),
		Close:         false,
	}
}

func NewResponse503(req *Request) *Response {
	html := "<html>" +
		"<head><title>503 Service Unavailable</title></head>\n" +
		"<body bgcolor=\"white\">\n" +
		"<center><h1>503 Service Unavailable</h1></center>\n" +
		"<hr><center>Go HTTP package</center>\n" +
		"</body></html>"
	return &Response{
		Status:        "Service Unavailable",
		StatusCode:    503,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Request:       req,
		Body:          NewBodyString(html),
		ContentLength: int64(len(html)),
		Close:         false,
	}
}

func NewResponse400(req *Request) *Response {
	html := "<html>" +
		"<head><title>400 Bad Request</title></head>\n" +
		"<body bgcolor=\"white\">\n" +
		"<center><h1>400 Bad Request</h1></center>\n" +
		"<hr><center>Go HTTP package</center>\n" +
		"</body></html>"
	return &Response{
		Status:        "Bad Request",
		StatusCode:    400,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Request:       req,
		Body:          NewBodyString(html),
		ContentLength: int64(len(html)),
		Close:         false,
	}
}

func NewResponse400String(req *Request, body string) *Response {
	return &Response{
		Status:        "Bad Request",
		StatusCode:    400,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Request:       req,
		Body:          NewBodyString(body),
		ContentLength: int64(len(body)),
		Close:         false,
	}
}

func NewResponse404(req *Request) *Response {
	html := "<html>" +
		"<head><title>404 Not found</title></head>\n" +
		"<body bgcolor=\"white\">\n" +
		"<center><h1>404 Not found</h1></center>\n" +
		"<hr><center>Go HTTP package</center>\n" +
		"</body></html>"
	return NewResponse404String(req, html)
}

func NewResponse404String(req *Request, s string) *Response {
	return &Response{
		Status:        "Not found",
		StatusCode:    404,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Request:       req,
		Body:          NewBodyString(s),
		ContentLength: int64(len(s)),
		Close:         false,
	}
}
