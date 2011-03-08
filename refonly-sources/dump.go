// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"bytes"
	"io"
	"os"
)

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

// StringToBody converts a string to an io.ReadCloser.
func StringToBody(s string) io.ReadCloser {
	b := bytes.NewBufferString(s)
	return NopCloser{b}
}

// One of the copies, say from b to r2, could be avoided by using a more
// elaborate trick where the other copy is made during Request/Response.Write.
// This would complicate things too much, given that these functions are for
// debugging only.
func drainBody(b io.ReadCloser) (r1, r2 io.ReadCloser, err os.Error) {
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, nil, err
	}
	if err = b.Close(); err != nil {
		return nil, nil, err
	}
	return NopCloser{&buf}, NopCloser{bytes.NewBuffer(buf.Bytes())}, nil
}

// DumpRequest returns the wire representation of req,
// optionally including the request body, for debugging.
// DumpRequest is semantically a no-op, but in order to
// dump the body, it reads the body data into memory and
// changes req.Body to refer to the in-memory copy.
func DumpRequest(req *Request, body bool) (dump []byte, err os.Error) {
	var b bytes.Buffer
	save := req.Body
	if !body || req.Body == nil {
		req.Body = nil
	} else {
		save, req.Body, err = drainBody(req.Body)
		if err != nil {
			return
		}
	}
	err = req.Write(&b)
	req.Body = save
	if err != nil {
		return
	}
	dump = b.Bytes()
	return
}

// DumpResponse is like DumpRequest but dumps a response.
func DumpResponse(resp *Response, body bool) (dump []byte, err os.Error) {
	var b bytes.Buffer
	save := resp.Body
	savecl := resp.ContentLength
	if !body || resp.Body == nil {
		resp.Body = nil
		resp.ContentLength = 0
	} else {
		save, resp.Body, err = drainBody(resp.Body)
		if err != nil {
			return
		}
	}
	err = resp.Write(&b)
	resp.Body = save
	resp.ContentLength = savecl
	if err != nil {
		return
	}
	dump = b.Bytes()
	return
}
