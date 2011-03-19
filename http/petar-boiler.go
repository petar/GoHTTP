// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

func NewResponse200() *Response {
	return &Response{
		Status:        "OK",
		StatusCode:    200,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		RequestMethod: "GET",
		Close:         false,
	}
}

func NewResponse200Bytes(b []byte) *Response {
	return &Response{
		Status:        "OK",
		StatusCode:    200,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		RequestMethod: "GET",
		Body:          NewBodyBytes(b),
		ContentLength: int64(len(b)),
		Close:         false,
	}
}

func NewResponse200CONNECT() *Response {
	return &Response{
		Status:        "Connection Established",
		StatusCode:    200,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		RequestMethod: "CONNECT",
		Close:         false,
		Header:        Header{"Proxy-Agent": []string{"Go-HTTP-package"}},
	}
}

func NewResponse500() *Response {
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
		RequestMethod: "GET",
		Body:          NewBodyString(html),
		ContentLength: int64(len(html)),
		Close:         false,
	}
}

func NewResponse503() *Response {
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
		RequestMethod: "GET",
		Body:          NewBodyString(html),
		ContentLength: int64(len(html)),
		Close:         false,
	}
}

func NewResponse400() *Response {
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
		RequestMethod: "GET",
		Body:          NewBodyString(html),
		ContentLength: int64(len(html)),
		Close:         false,
	}
}

func NewResponse400String(body string) *Response {
	return &Response{
		Status:        "Bad Request",
		StatusCode:    400,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		RequestMethod: "GET",
		Body:          NewBodyString(body),
		ContentLength: int64(len(body)),
		Close:         false,
	}
}

func NewResponse404() *Response {
	html := "<html>" +
		"<head><title>404 Not found</title></head>\n" +
		"<body bgcolor=\"white\">\n" +
		"<center><h1>404 Not found</h1></center>\n" +
		"<hr><center>Go HTTP package</center>\n" +
		"</body></html>"
	return NewResponse404String(html)
}

func NewResponse404String(s string) *Response {
	return &Response{
		Status:        "Not found",
		StatusCode:    404,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		RequestMethod: "GET",
		Body:          NewBodyString(s),
		ContentLength: int64(len(s)),
		Close:         false,
	}
}
