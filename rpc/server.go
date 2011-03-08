// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"log"
	"os"
	"reflect"
	"sync"
	"unicode"
	"utf8"
)

// Precompute the reflect type for os.Error.  Can't use os.Error directly
// because Typeof takes an empty interface value.  This is annoying.
var unusedError *os.Error
var typeOfOsError = reflect.Typeof(unusedError).(*reflect.PtrType).Elem()

type methodType struct {
	sync.Mutex // protects counters
	method     reflect.Method
	ArgType    *reflect.PtrType
	ReplyType  *reflect.PtrType
	numCalls   uint
}

type service struct {
	name   string                 // name of service
	rcvr   reflect.Value          // receiver of methods for the service
	typ    reflect.Type           // type of the receiver
	method map[string]*methodType // registered methods
}

// Request is a header written before every RPC call.  It is used internally
// but documented here as an aid to debugging, such as when analyzing
// network traffic.
type Request struct {
	ServiceMethod string // format: "Service.Method"
	Seq           uint64 // sequence number chosen by client
}

// Response is a header written before every RPC return.  It is used internally
// but documented here as an aid to debugging, such as when analyzing
// network traffic.
type Response struct {
	ServiceMethod string // echoes that of the Request
	Seq           uint64 // echoes that of the request
	Error         string // error, if any.
}

// Server represents an RPC Server.
type Server struct {
	sync.Mutex // protects the serviceMap
	serviceMap map[string]*service
}

// NewServer returns a new Server.
func NewServer() *Server {
	return &Server{serviceMap: make(map[string]*service)}
}

// Is this an exported - upper case - name?
func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

// Register publishes in the server the set of methods of the
// receiver value that satisfy the following conditions:
//	- exported method
//	- two arguments, both pointers to exported structs
//	- one return value, of type os.Error
// It returns an error if the receiver is not an exported type or has no
// suitable methods.
// The client accesses each method using a string of the form "Type.Method",
// where Type is the receiver's concrete type.
func (server *Server) Register(rcvr interface{}) os.Error {
	return server.register(rcvr, "", false)
}

// RegisterName is like Register but uses the provided name for the type 
// instead of the receiver's concrete type.
func (server *Server) RegisterName(name string, rcvr interface{}) os.Error {
	return server.register(rcvr, name, true)
}

func (server *Server) register(rcvr interface{}, name string, useName bool) os.Error {
	server.Lock()
	defer server.Unlock()
	if server.serviceMap == nil {
		server.serviceMap = make(map[string]*service)
	}
	s := new(service)
	s.typ = reflect.Typeof(rcvr)
	s.rcvr = reflect.NewValue(rcvr)
	sname := reflect.Indirect(s.rcvr).Type().Name()
	if useName {
		sname = name
	}
	if sname == "" {
		log.Fatal("rpc: no service name for type", s.typ.String())
	}
	if s.typ.PkgPath() != "" && !isExported(sname) && !useName {
		s := "rpc Register: type " + sname + " is not exported"
		log.Print(s)
		return os.ErrorString(s)
	}
	if _, present := server.serviceMap[sname]; present {
		return os.ErrorString("rpc: service already defined: " + sname)
	}
	s.name = sname
	s.method = make(map[string]*methodType)

	// Install the methods
	for m := 0; m < s.typ.NumMethod(); m++ {
		method := s.typ.Method(m)
		mtype := method.Type
		mname := method.Name
		if mtype.PkgPath() != "" || !isExported(mname) {
			continue
		}
		// Method needs three ins: receiver, *args, *reply.
		if mtype.NumIn() != 3 {
			log.Println("method", mname, "has wrong number of ins:", mtype.NumIn())
			continue
		}
		argType, ok := mtype.In(1).(*reflect.PtrType)
		if !ok {
			log.Println(mname, "arg type not a pointer:", mtype.In(1))
			continue
		}
		replyType, ok := mtype.In(2).(*reflect.PtrType)
		if !ok {
			log.Println(mname, "reply type not a pointer:", mtype.In(2))
			continue
		}
		if argType.Elem().PkgPath() != "" && !isExported(argType.Elem().Name()) {
			log.Println(mname, "argument type not exported:", argType)
			continue
		}
		if replyType.Elem().PkgPath() != "" && !isExported(replyType.Elem().Name()) {
			log.Println(mname, "reply type not exported:", replyType)
			continue
		}
		// Method needs one out: os.Error.
		if mtype.NumOut() != 1 {
			log.Println("method", mname, "has wrong number of outs:", mtype.NumOut())
			continue
		}
		if returnType := mtype.Out(0); returnType != typeOfOsError {
			log.Println("method", mname, "returns", returnType.String(), "not os.Error")
			continue
		}
		s.method[mname] = &methodType{method: method, ArgType: argType, ReplyType: replyType}
	}

	if len(s.method) == 0 {
		s := "rpc Register: type " + sname + " has no exported methods of suitable type"
		log.Print(s)
		return os.ErrorString(s)
	}
	server.serviceMap[s.name] = s
	return nil
}

// A value sent as a placeholder for the response when the server receives an invalid request.
type InvalidRequest struct {
	Marker int
}
var invalidRequest = InvalidRequest{}

func _new(t *reflect.PtrType) *reflect.PtrValue {
	v := reflect.MakeZero(t).(*reflect.PtrValue)
	v.PointTo(reflect.MakeZero(t.Elem()))
	return v
}

/*
func sendResponse(sending *sync.Mutex, req *Request, reply interface{}, codec ServerCodec, errmsg string) {
	resp := new(Response)
	// Encode the response header
	resp.ServiceMethod = req.ServiceMethod
	if errmsg != "" {
		resp.Error = errmsg
		reply = invalidRequest
	}
	resp.Seq = req.Seq
	sending.Lock()
	err := codec.WriteResponse(resp, reply)
	if err != nil {
		log.Println("rpc: writing response:", err)
	}
	sending.Unlock()
}
*/

func (m *methodType) NumCalls() (n uint) {
	m.Lock()
	n = m.numCalls
	m.Unlock()
	return n
}

/*
func (s *service) call(sending *sync.Mutex, mtype *methodType, req *Request, argv, replyv reflect.Value, codec ServerCodec) {
	mtype.Lock()
	mtype.numCalls++
	mtype.Unlock()
	function := mtype.method.Func
	// Invoke the method, providing a new value for the reply.
	returnValues := function.Call([]reflect.Value{s.rcvr, argv, replyv})
	// The return value for the method is an os.Error.
	errInter := returnValues[0].Interface()
	errmsg := ""
	if errInter != nil {
		errmsg = errInter.(os.Error).String()
	}
	sendResponse(sending, req, replyv.Interface(), codec, errmsg)
}
*/

// ServeCodec is like ServeConn but uses the specified codec to
// decode requests and encode responses.
/*
func (server *Server) ServeCodec(codec ServerCodec) {
	sending := new(sync.Mutex)
	for {
		req, service, mtype, err := server.readRequest(codec)
		if err != nil {
			if err != os.EOF {
				log.Println("rpc:", err)
			}
			if err == os.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			// discard body
			codec.ReadRequestBody(nil)

			// send a response if we actually managed to read a header.
			if req != nil {
				sendResponse(sending, req, invalidRequest, codec, err.String())
			}
			continue
		}

		// Decode the argument value.
		argv := _new(mtype.ArgType)
		replyv := _new(mtype.ReplyType)
		err = codec.ReadRequestBody(argv.Interface())
		if err != nil {
			if err == os.EOF || err == io.ErrUnexpectedEOF {
				if err == io.ErrUnexpectedEOF {
					log.Println("rpc:", err)
				}
				break
			}
			sendResponse(sending, req, replyv.Interface(), codec, err.String())
			continue
		}
		go service.call(sending, mtype, req, argv, replyv, codec)
	}
	codec.Close()
}
*/
/*
func (server *Server) readRequest(codec ServerCodec) (req *Request, service *service, mtype *methodType, err os.Error) {
	// Grab the request header.
	req = new(Request)
	err = codec.ReadRequestHeader(req)
	if err != nil {
		req = nil
		if err == os.EOF || err == io.ErrUnexpectedEOF {
			return
		}
		err = os.ErrorString("rpc: server cannot decode request: " + err.String())
		return
	}

	serviceMethod := strings.Split(req.ServiceMethod, ".", -1)
	if len(serviceMethod) != 2 {
		err = os.ErrorString("rpc: service/method request ill-formed: " + req.ServiceMethod)
		return
	}
	// Look up the request.
	server.Lock()
	service = server.serviceMap[serviceMethod[0]]
	server.Unlock()
	if service == nil {
		err = os.ErrorString("rpc: can't find service " + req.ServiceMethod)
		return
	}
	mtype = service.method[serviceMethod[1]]
	if mtype == nil {
		err = os.ErrorString("rpc: can't find method " + req.ServiceMethod)
	}
	return
}
*/

// A ServerCodec implements reading of RPC requests and writing of
// RPC responses for the server side of an RPC session.
// The server calls ReadRequestHeader and ReadRequestBody in pairs
// to read requests from the connection, and it calls WriteResponse to
// write a response back.  The server calls Close when finished with the
// connection. ReadRequestBody may be called with a nil
// argument to force the body of the request to be read and discarded.
type ServerCodec interface {
	ReadRequestHeader(*Request) os.Error
	ReadRequestBody(interface{}) os.Error
	WriteResponse(*Response, interface{}) os.Error

	Close() os.Error
}
