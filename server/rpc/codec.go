// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"json"
	"os"
	"path"
	"rpc"
	"strings"
	"github.com/petar/GoHTTP/http"
	"github.com/petar/GoHTTP/server"
)


// httpCodec is an rpc.ServerCodec for the RPC server
// It parses an incoming HTTP request into a an RPC arguments variable
// that has the structure described above.
type queryCodec struct {
	*server.Query

	// seq is not protected by a mutex because it is accessed only inside
	// the read methods, which are guaranteed to be called sequentially
	// by rpc.Server
	seq uint64
}

var ErrCodec = os.NewError("http/rpc codec")

// rpc.Server calls ReadRequestHeader and ReadRequestBody in a 
// synchronous sequence. If ReadRequestHeader returns an error,
// then ReadRequestBody is either not called (if err == os.EOF or 
// err == io.ErrUnexpectedEOF), or called with a nil argument 
// (for any other err). WriteResponse is called out of sync, and
// only if ReadRequestBody returns no error.

func (qx *queryCodec) ReadRequestHeader(req *rpc.Request) os.Error {
	if qx.seq == 0 {
		return os.EOF
	}
	req.Seq = qx.seq
	req.ServiceMethod = pathToServiceMethod(qx.Req.URL.Path)
	return nil
}

func pathToServiceMethod(p string) string {
	p = path.Clean(p)
	if p != "" && p[0] == '/' {
		p = p[1:]
	}
	return strings.Replace(p, "/", ".", -1)
}

// ReadRequestBody parses the URL for the AJAX parameters
func (qx *queryCodec) ReadRequestBody(args interface{}) (err os.Error) {
	defer func() {
		qx.seq = 0
	}()
	if args == nil {
		if qx.Query.Req.Body != nil {
			qx.Query.Req.Body.Close()
		}
		return nil
	}

	a := args.(*Args)

	// Save request method (GET, POST, PUT, UPDATE, etc.)
	a.Method = qx.Query.Req.Method

	// Decode URL arguments
	a.Query, err = http.ParseQuery(qx.Query.Req.URL.RawQuery)
	if err != nil {
		return err
	}

	// Decode JSON body
	a.Body = make(map[string]interface{})
	if qx.Query.Req.Body != nil {
		dec := json.NewDecoder(qx.Query.Req.Body)
		// We don't care if the decode is successful.
		// The user will do their own complaining if they are missing expected arguments.
		dec.Decode(a.Body)
		qx.Query.Req.Body.Close()
	}

	// Read the cookies associated with the request
	a.Cookies = qx.Query.Req.Cookies()

	return nil
}

func (qx *queryCodec) WriteResponse(resp *rpc.Response, ret interface{}) (err os.Error) {

	if resp.Error != "" {
		return qx.Query.Write(http.NewResponse400String(qx.Query.Req, resp.Error))
	}

	if ret == nil {
		return qx.Query.Write(http.NewResponse200(qx.Query.Req))
	}

	r := ret.(*Ret)

	var body []byte
	if r.Value != nil {
		body, err = json.Marshal(r.Value)
		if err != nil {
			qx.Query.Write(http.NewResponse500(qx.Query.Req))
			return err
		}
	}

	httpResp := http.NewResponse200Bytes(qx.Query.Req, body)
	httpResp.Header = make(http.Header)
	for _, setCookie := range r.SetCookies {
		httpResp.Header.Add("Set-Cookie", setCookie.String())
	}

	return qx.Query.Write(httpResp)
}

func (qx *queryCodec) Close() os.Error { return nil }
