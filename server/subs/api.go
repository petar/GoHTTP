// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package subs

import (
	"json"
	"os"
	"path"
	"reflect"
	"rpc"
	"strconv"
	"strings"
	"sync"
	"github.com/petar/GoHTTP/http"
	"github.com/petar/GoHTTP/server"
)

// APISub is a Sub that acts as an HTTP RPC server.
// Requests are received in the form of HTTP GET requests with
// parameters in the URL, just like the ones produced by jQuery's
// AJAX calls. Responses are returned in the form of HTTP responses
// with return values in the form of a JSON object in the response
// body.
type APISub struct {
	rpc *rpc.Server // does not need locking, since re-entrant
	sync.Mutex  // protects auto
	auto  uint64
}

func NewAPISub() *APISub {
	return &APISub{
		rpc:   rpc.NewServer(),
		auto:  1, // Start seq numbers from 1, so that 0 is always an invalid seq number
	}
}

func (api *APISub) Register(rcvr interface{}) os.Error {
	return api.rpc.Register(rcvr)
}

func (api *APISub) RegisterName(name string, rcvr interface{}) os.Error {
	return api.rpc.RegisterName(name, rcvr)
}

func (api *APISub) Serve(q *server.Query) {
	qx := &queryCodec{ Query: q }
	api.Lock()
	qx.seq = api.auto
	api.auto++
	api.Unlock()
	q.Continue()
	go api.rpc.ServeCodec(qx)
}

// httpCodec is an rpc.ServerCodec for the APISub server
type queryCodec struct {
	*server.Query
	seq uint64
}

// rpc.Server calls ReadRequestHeader and ReadRequestBody in a 
// synchronous sequence. If ReadRequestHeader returns an error,
// then ReadRequestBody is either not called (if err == os.EOF or 
// err == io.ErrUnexpectedEOF), or called with a nil argument 
// (for any other err). WriteResponse is called out of sync., and
// only if ReadRequestBody returns no error.

func (qx *queryCodec) ReadRequestHeader(req *rpc.Request) os.Error {
	if qx.Query == nil {
		return os.EOF
	}
	if qx.Query.Req.Body != nil {
		qx.Query.Req.Body.Close() // Discard HTTP body. Only GET requests supported currently.
	}
	req.Seq = qx.seq
	req.ServiceMethod = pathToServiceMethod(qx.Req.URL.Path)
	return nil
}

// ReadRequestBody parses the URL for the AJAX parameters
func (qx *queryCodec) ReadRequestBody(body interface{}) os.Error {
	if body == nil {
		return nil
	}
	bmap, err := http.ParseQuery(qx.Query.Req.URL.RawQuery)
	if err != nil {
		return err
	}
	return decodeMap(bmap, body)
}

func (qx *queryCodec) WriteResponse(resp *rpc.Response, body interface{}) os.Error {
	defer func(){ qx.Query = nil }()

	if resp.Error != "" {
		return qx.Query.Write(http.NewResponse400String(resp.Error))
	}
	buf, err := json.Marshal(body) 
	if err != nil {
		qx.Query.Write(http.NewResponse500())
		return ErrCodec
	}
	return qx.Query.Write(http.NewResponse200Bytes(buf))
}

func (qx *queryCodec) Close() os.Error { return nil }

func pathToServiceMethod(p string) string {
	p = path.Clean(p)
	if p != "" && p[0] == '/' {
		p = p[1:]
	}
	return strings.Replace(p, "/", ".", -1)
}

var ErrCodec = os.NewError("api codec")

// TODO: Maybe add logic to parse array/slice values
func decodeMap(m map[string][]string, v interface{}) os.Error {

	rv := reflect.NewValue(v)

	// If the user wants result in the form of a map, just copy the contents
	mv, ok := rv.(*reflect.MapValue)
	if ok {
		mv.Set(reflect.NewValue(m).(*reflect.MapValue))
		return nil
	}

	// Otherwise, we expect a pointer to a non-recursive struct
	pv, ok := rv.(*reflect.PtrValue)
	if !ok || pv.IsNil() {
		return ErrCodec
	}
	sv, ok := pv.Elem().(*reflect.StructValue)
	if !ok {
		return ErrCodec
	}

	for k, ss := range m {
		if len(ss) == 0 {
			continue
		}
		fv := sv.FieldByName(k)
		if fv == nil {
			continue
		}
		switch fv := fv.(type) {
		case *reflect.BoolValue:
			i, err := strconv.Atoi(ss[0])
			if err != nil || i < 0 {
				return ErrCodec
			}
			fv.Set(i > 0)

		case *reflect.FloatValue:
			f, err := strconv.Atof64(ss[0])
			if err != nil {
				return ErrCodec
			}
			fv.Set(f)

		case *reflect.IntValue:
			i, err := strconv.Atoi64(ss[0])
			if err != nil {
				return ErrCodec
			}
			fv.Set(i)

		case *reflect.StringValue:
			fv.Set(ss[0])

		default:
			continue
		}
	}

	return nil
}

