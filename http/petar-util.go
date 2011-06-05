// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"github.com/petar/GoGauge/unclosed"
)

// NewBodyString converts a string to an io.ReadCloser.
func NewBodyString(s string) io.ReadCloser { return ioutil.NopCloser(bytes.NewBufferString(s)) }

// NewBodyBytes converts a byte slice to an io.ReadCloser.
func NewBodyBytes(b []byte) io.ReadCloser { return ioutil.NopCloser(bytes.NewBuffer(b)) }

func NewBodyFile(filename string) (io.ReadCloser, os.Error) {
	f, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return unclosed.NewReadCloserTracker(ioutil.NopCloser(bytes.NewBuffer(f))), nil
}

func NewResponseFile(req *Request, filename string) (*Response, os.Error) {
	b, err := NewBodyFile(filename)
	if err != nil {
		return NewResponse404(req), err
	}
	r := NewResponse200(req)
	r.Body = b
	r.TransferEncoding = []string{"chunked"}
	r.ContentLength = -1
	return r, nil
}

func NewResponseWithBody(req *Request, r io.ReadCloser) *Response {
	resp := NewResponse200(req)
	resp.Body = ioutil.NopCloser(r)
	resp.TransferEncoding = []string{"chunked"}
	resp.ContentLength = -1
	return resp
}

func NewResponseWithBytes(req *Request, b []byte) *Response {
	return NewResponseWithBody(req, NewBodyBytes(b))
}

func NewResponseWithReader(req *Request, r io.Reader) *Response {
	return NewResponseWithBody(req, ioutil.NopCloser(r))
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
