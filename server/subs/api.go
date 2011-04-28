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

// API is a Sub that acts as an HTTP RPC server.
// Requests are received in the form of HTTP GET requests with
// parameters in the URL, just like the ones produced by jQuery's
// AJAX calls. Responses are returned in the form of HTTP responses
// with return values in the form of a JSON object in the response
// body.
type API struct {
	rpcs       *rpc.Server // does not need locking, since re-entrant
	sync.Mutex             // protects auto
	auto       uint64
}

func NewAPI() *API {
	return &API{
		rpcs: rpc.NewServer(),
		auto: 1, // Start seq numbers from 1, so that 0 is always an invalid seq number
	}
}

func (api *API) Register(rcvr interface{}) os.Error {
	return api.rpcs.Register(rcvr)
}

func (api *API) RegisterName(name string, rcvr interface{}) os.Error {
	return api.rpcs.RegisterName(name, rcvr)
}

func (api *API) Serve(q *server.Query) {
	qx := &queryCodec{Query: q}
	api.Lock()
	qx.seq = api.auto
	api.auto++
	api.Unlock()
	q.Continue()
	go api.rpcs.ServeCodec(qx)
}

// httpCodec is an rpc.ServerCodec for the API server
type queryCodec struct {
	*server.Query

	// seq is not protected by a mutex because it is accessed only inside
	// the read methods, which are guaranteed to be called sequentially
	// by rpc.Server
	seq uint64
}

// rpc.Server calls ReadRequestHeader and ReadRequestBody in a 
// synchronous sequence. If ReadRequestHeader returns an error,
// then ReadRequestBody is either not called (if err == os.EOF or 
// err == io.ErrUnexpectedEOF), or called with a nil argument 
// (for any other err). WriteResponse is called out of sync., and
// only if ReadRequestBody returns no error.

func (qx *queryCodec) ReadRequestHeader(req *rpc.Request) os.Error {
	if qx.seq == 0 {
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
	defer func() {
		qx.seq = 0
	}()
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

	vv := reflect.ValueOf(v)

	// If the user wants result in the form of a map, just copy the contents
	if vv.Type().Kind() == reflect.Map {
		vv.Set(reflect.ValueOf(m))
		return nil
	}

	// Otherwise, we expect a pointer to a non-recursive struct
	if vv.Type().Kind() != reflect.Ptr || vv.IsNil() {
		return ErrCodec
	}
	if vv.Elem().Type().Kind() != reflect.Struct {
		return ErrCodec
	}
	sv := vv.Elem()

	for k, ss := range m {
		if len(ss) == 0 {
			continue
		}
		fv := sv.FieldByName(k)
		if !fv.IsValid() {
			continue
		}
		switch fv.Type().Kind() {
		case reflect.Bool:
			i, err := strconv.Atoi(ss[0])
			if err != nil || i < 0 {
				return ErrCodec
			}
			fv.SetBool(i > 0)

		case reflect.Float32, reflect.Float64:
			f, err := strconv.Atof64(ss[0])
			if err != nil {
				return ErrCodec
			}
			fv.SetFloat(f)

		case reflect.Int:
			i, err := strconv.Atoi64(ss[0])
			if err != nil {
				return ErrCodec
			}
			fv.SetInt(i)

		case reflect.String:
			fv.SetString(ss[0])

		default:
			continue
		}
	}

	return nil
}
