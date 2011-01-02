// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
)

// NopCloser adds a no-op Close method to a Reader object to
// convert into an io.ReadCloser. This is handy when you need to
// use e.g. a bytes.Buffer buf as a Body. In this case you
// would Request.Body = NopCloser{buf}
type NopCloser struct {
	io.Reader
}

func (NopCloser) Close() os.Error { return nil }

// NewBodyString converts a string to an io.ReadCloser.
func NewBodyString(s string) io.ReadCloser {
	b := bytes.NewBufferString(s)
	return NopCloser{b}
}

func NewBodyFile(filename string) (io.ReadCloser, os.Error) {
	f, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return NopCloser{bytes.NewBuffer(f)}, nil
}

func NewResponseFile(filename string) (*Response, os.Error) {
	b,err := NewBodyFile(filename)
	if err != nil {
		return NewResponse404(), err
	}
	r := NewResponse200()
	r.Body = b
	r.TransferEncoding = []string{"chunked"}
	r.ContentLength = -1
	return r, nil
}

// DupResp returns a replica of resp and any error encountered
// while reading resp.Body.
func DupResp(resp *Response) (r2 *Response, err os.Error) {
	tmp := *resp
	if resp.Body != nil {
		resp.Body, tmp.Body, err = drainBody(resp.Body)
	}
	if err != nil {
		return nil, err
	}
	return &tmp, err
}

// DupReq returns a replica of req and any error encountered
// while reading req.Body.
func DupReq(req *Request) (r2 *Request, err os.Error) {
	tmp := *req
	if req.Body != nil {
		req.Body, tmp.Body, err = drainBody(req.Body)
	}
	if err != nil {
		return nil, err
	}
	return &tmp, err
}
