// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"json"
	"os"
	"path"
	"reflect"
	"rpc"
	"strconv"
	"strings"
	"github.com/petar/GoHTTP/http"
	"github.com/petar/GoHTTP/server"
)

//  XXX: document
//
//  Possible types of the argument structure's fields Args and Cookies:
//
//   Cookies []*Cookie
//   Args    struct_type
//           ptr_to_struct_type
//           map[string][]string
//           map[string]string

type LongCookieArgs struct {
	Cookies []*http.Cookie
	Args    map[string][]string
}

type ShortCookieArgs struct {
	Cookies []*http.Cookie
	Args    map[string]string
}

type LongArgs struct {
	Args    map[string][]string
}

type ShortArgs struct {
	Args    map[string]string
}

type CookieArgs struct {
	Cookies []*http.Cookie
}

type NoArgs struct {}

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

var ErrCodec = os.NewError("api codec")

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
	if qx.Query.Req.Body != nil {
		qx.Query.Req.Body.Close() // Discard HTTP body. Only GET requests supported currently.
	}
	req.Seq = qx.seq
	req.ServiceMethod = pathToServiceMethod(qx.Req.URL.Path)
	return nil
}

// ReadRequestBody parses the URL for the AJAX parameters
func (qx *queryCodec) ReadRequestBody(args interface{}) os.Error {
	defer func() {
		qx.seq = 0
	}()
	if args == nil {
		return nil
	}

	// Parse the arguments structure
	av := reflect.ValueOf(args)

	// If args is non-nil, it must be a pointer to struct that has any subset of the fields
	// Cookies and Args
	if av.Type().Kind() != reflect.Ptr {
		return ErrCodec
	}
	if av.Elem().Type().Kind() != reflect.Struct {
		return ErrCodec
	}
	sv := av.Elem()

	// Parse URL arguments
	// We expect that the field Args (if present) is one of:
	// (*) struct, (*) pointer to struct, (*) map[string][]string, or (*) map[string]string
	uv := sv.FieldByName("Args")
	if uv.IsValid() {
		mm, err := http.ParseQuery(qx.Query.Req.URL.RawQuery)
		if err != nil {
			return err
		}
		switch uv.Type().Kind() {

		// struct
		case reflect.Struct:
			return decodeMapToNonRecursiveStruct(mm, uv)

		// *struct
		case reflect.Ptr:
			ev := uv.Elem()
			if ev.Type().Kind() != reflect.Struct {
				return ErrCodec
			}
			return decodeMapToNonRecursiveStruct(mm, ev)

		// map[string]string or map[string][]string
		case reflect.Map:
			mt := uv.Type()
			if mt.Key().Kind() != reflect.String {
				return ErrCodec
			}
			et := mt.Elem()
			switch et.Kind() {
			case reflect.String:
				uv.Set(reflect.ValueOf(simplifyMap(mm)))
			case reflect.Slice:
				if et.Elem().Kind() != reflect.String {
					return ErrCodec
				}
				uv.Set(reflect.ValueOf(mm))
			default:
				return ErrCodec
			}
		default:
			return ErrCodec
		}
	}

	// Parse Cookie arguments
	cv := sv.FieldByName("Cookies")
	if cv.IsValid() {
		cv.Set(reflect.ValueOf(qx.Query.Req.Cookies()))
	}

	return nil
}

func simplifyMap(mm map[string][]string) map[string]string {
	m := make(map[string]string)
	for k, v := range mm {
		if len(v) > 0 {
			m[k] = v[0]
		}
	}
	return m
}

func decodeMapToNonRecursiveStruct(m map[string][]string, sv reflect.Value) os.Error {

	if sv.Type().Kind() != reflect.Struct {
		return ErrCodec
	}

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

func (qx *queryCodec) WriteResponse(resp *rpc.Response, body interface{}) os.Error {
	if resp.Error != "" {
		return qx.Query.Write(http.NewResponse400String(qx.Query.Req, resp.Error))
	}
	buf, err := json.Marshal(body)
	if err != nil {
		qx.Query.Write(http.NewResponse500(qx.Query.Req))
		return ErrCodec
	}
	return qx.Query.Write(http.NewResponse200Bytes(qx.Query.Req, buf))
}

func (qx *queryCodec) Close() os.Error { return nil }

func pathToServiceMethod(p string) string {
	p = path.Clean(p)
	if p != "" && p[0] == '/' {
		p = p[1:]
	}
	return strings.Replace(p, "/", ".", -1)
}
