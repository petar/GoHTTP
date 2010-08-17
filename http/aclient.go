// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"container/list"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// An AsyncClient fetches responses to HTTP requests. Internally,
// it maintains a set of keep-alive connections to remote
// destinations and tries to reuse them. When a Fetch() request
// is made, the request is sent to the desired remote host over
// an existing connection or a new TCP connection is established
// as needed.
//
// TODO(petar): Eventually, AsyncClient will allow for a user-specified
// mechanism for establishing new connections, so that e.g. it could
// be asked to go through a proxy.
type AsyncClient struct {
	tmo      int64              // keepalive timout
	maxpiped int                // maximum pipelined requests per AsyncClientConn
	maxatt   int                // maximum number of retries
	hostmap  map[string]*remote // "host:port" -> remote
	lk       sync.Mutex
	fdl      FDLimiter
	shut     bool
}

// A remote struct holds all connections to the same remote host.
type remote struct {
	connlist list.List // a list of active AsyncClientConn's
}

type stampedClientConn struct {
	*AsyncClientConn
	stamp int64
	lk    sync.Mutex
}

func newStampedClientConn(c net.Conn) *stampedClientConn {
	return &stampedClientConn{
		AsyncClientConn: NewAsyncClientConn(c),
		stamp:           time.Nanoseconds(),
	}
}

func (scc *stampedClientConn) touch() {
	scc.lk.Lock()
	defer scc.lk.Unlock()
	scc.stamp = time.Nanoseconds()
}

func (scc *stampedClientConn) GetStamp() int64 {
	scc.lk.Lock()
	defer scc.lk.Unlock()
	return scc.stamp
}

func (scc *stampedClientConn) Fetch(req *Request) (resp *Response, err os.Error) {
	scc.touch()
	defer scc.touch()
	return scc.AsyncClientConn.Fetch(req)
}

// NewAsyncClient creates a new AsyncClient object.
//
// tmo specifies the HTTP keep-alive timeout for outgoing connections.
// maxpiped specifies the maximum number of pipelined requests
// sent over any given connection;
// maxatt specifies the maximum number of retries for any given request;
// fdlim specifies the maximum number of file descriptors that can be
// utilized at any given time;
func NewAsyncClient(tmo int64, maxpiped, maxatt, fdlim int) *AsyncClient {
	ac := &AsyncClient{
		tmo:      tmo,
		maxatt:   maxatt,
		maxpiped: maxpiped,
		hostmap:  make(map[string]*remote),
	}
	ac.fdl.Init(fdlim)
	go ac.expireLoop()
	return ac
}

func (ac *AsyncClient) GetFDLimiter() *FDLimiter { return &ac.fdl }

func attachPort(host string, defaultPort int) string {
	if strings.Index(host, ":") < 0 {
		return host + ":" + strconv.Itoa(defaultPort)
	}
	return host
}

// Connect initiates a CONNECT request, according to the request req.
// resp is the response to the CONNECT request, it is never nil;
// conn is the established connection or nil, in case of error;
func (ac *AsyncClient) Connect(req *Request) (resp *Response, conn net.Conn) {

	host := req.Host
	if host == "" {
		host = req.URL.Host
	}
	if host == "" {
		if req.Body != nil {
			req.Body.Close()
		}
		return newRespBadRequest(), nil
	}
	host = attachPort(host, 443)

	if req.Body != nil {
		req.Body.Close()
	}
	for attempt := 0; attempt < ac.maxatt; attempt++ {
		ac.reclaim()
		if ac.fdl.LockOrTimeout(60e9) != nil {
			break
		}
		conn, _ = net.Dial("tcp", "", host)
		if conn != nil {
			rocConn := NewConnRunOnClose(conn, func() { ac.fdl.Unlock() })
			return respConnectionEstablished, rocConn
		}
		ac.fdl.Unlock()
	}
	// Cannot connect
	return newRespServiceUnavailable(), nil
}

// Fix the host-to-host part of the request
func fixRequest(req *Request) {
	req.Proto = "HTTP/1.1"
	req.ProtoMajor = 1
	req.ProtoMinor = 1
	req.Close = false
	req.Header["Connection"] = "", false
}

// Fetch initiates attempts to obtain a response to the request req.
// TODO(petar): Perhaps, return an os.Error describing the problem,
// instead of errors wrapped in HTTP responses.
func (ac *AsyncClient) Fetch(req *Request) *Response {

	fixRequest(req)
	host := req.Host
	if host == "" {
		host = req.URL.Host
	}
	if host == "" {
		if req.Body != nil {
			req.Body.Close()
		}
		return newRespBadRequest()
	}
	host = attachPort(host, 80)

	if req.Body != nil {
		req.Body = NewRewindReadCloser(req.Body, 1024)
	}
	for i := 0; i < ac.maxatt; i++ {
		scc := ac.findLightlyLoaded(host)
		if scc == nil {
			scc = ac.dial(host)
		}
		if scc == nil {
			sleepOnAttempt(i)
			continue
		}
		resp, _ := scc.Fetch(req)
		if resp != nil {
			return resp
		}

		// TODO(petar): Ideally, one differentiates between errors occuring
		// during the write- or read- stage of a Fetch(). Write-side errors
		// may not prompt a connection closure, if the read side is still empty
		// and more responses are awaited. But this is a rare scenario, so for
		// the sake of simplicity just kill the connection if the Fetch() failed.
		ac.bury(host, scc)

		if req.Body != nil {
			if req.Body.(*RewindReadCloser).Rewind() != nil {
				break
			}
		}
		sleepOnAttempt(i)
	}
	if req.Body != nil {
		req.Body.Close()
	}
	return newRespServiceUnavailable()

}

func sleepOnAttempt(i int) {
	if i == 0 {
		return
	}
	time.Sleep(1e9 << uint(i-1))
}

func (ac *AsyncClient) findLightlyLoaded(host string) (scc *stampedClientConn) {
	ac.lk.Lock()
	defer ac.lk.Unlock()
	r, ok := ac.hostmap[host]
	if !ok {
		return
	}
	elm := r.connlist.Front()
	for elm != nil {
		v := elm.Value.(*stampedClientConn)
		if v.Pending() < ac.maxpiped {
			scc = v
			break
		}
		elm = elm.Next()
	}
	return
}

func (ac *AsyncClient) dial(host string) *stampedClientConn {
	ac.reclaim()
	if ac.fdl.LockOrTimeout(60e9) != nil {
		return nil
	}
	conn, err := net.Dial("tcp", "", host)
	if err != nil {
		ac.fdl.Unlock()
		return nil
	}
	conn.(*net.TCPConn).SetKeepAlive(true)
	err = conn.SetTimeout(ac.tmo)
	if err != nil {
		ac.fdl.Unlock()
		return nil
	}
	conn = NewConnRunOnClose(conn, func() { ac.fdl.Unlock() })
	scc := newStampedClientConn(conn)

	ac.lk.Lock()
	r := ac.getOrMakeRemote(host)
	r.connlist.PushFront(scc)
	ac.lk.Unlock()
	return scc
}

func (ac *AsyncClient) reclaim() {
	used := ac.fdl.LockCount()
	lim := ac.fdl.Limit()
	if 5*used < 4*lim {
		return
	}
	var found *stampedClientConn
	ac.lk.Lock()
	for h, r := range ac.hostmap {
		elm := r.connlist.Front()
		for elm != nil {
			ssc := elm.Value.(*stampedClientConn)
			if ssc.Pending() == 0 {
				found = ssc
				r.connlist.Remove(elm)
				if r.connlist.Len() == 0 {
					ac.hostmap[h] = nil, false
				}
				goto __FoundIdle
			}
			elm = elm.Next()
		}
	}
__FoundIdle:
	ac.lk.Unlock()
	if found != nil {
		c, _, _ := found.Close()
		if c != nil {
			c.Close()
		}
	}
}

func (ac *AsyncClient) bury(host string, scc *stampedClientConn) {
	ac.lk.Lock()
	if r, ok := ac.hostmap[host]; ok {
		elm := r.connlist.Front()
		for elm != nil {
			v := elm.Value.(*stampedClientConn)
			if v == scc {
				r.connlist.Remove(elm)
				break
			}
			elm = elm.Next()
		}
		if r.connlist.Len() == 0 {
			ac.hostmap[host] = nil, false
		}
	}
	ac.lk.Unlock()
	conn, _, _ := scc.Close()
	if conn != nil {
		conn.Close()
	}
}

func (ac *AsyncClient) getOrMakeRemote(host string) *remote {
	r, ok := ac.hostmap[host]
	if ok {
		return r
	}
	r = &remote{}
	ac.hostmap[host] = r
	return r
}

type hostAndScc struct {
	host string
	scc  *stampedClientConn
}

func (ac *AsyncClient) expireLoop() {
	for {
		ac.lk.Lock()
		if ac.shut {
			ac.lk.Unlock()
			return
		}
		now := time.Nanoseconds()
		kills := list.New()
		for h, r := range ac.hostmap {
			elm := r.connlist.Front()
			for elm != nil {
				scc := elm.Value.(*stampedClientConn)
				if now-scc.GetStamp() >= ac.tmo {
					kills.PushBack(hostAndScc{h, scc})
				}
				elm = elm.Next()
			}
		}
		ac.lk.Unlock()
		elm := kills.Front()
		for elm != nil {
			t := elm.Value.(hostAndScc)
			ac.bury(t.host, t.scc)
			elm = elm.Next()
		}
		kills.Init()
		kills = nil
		time.Sleep(ac.tmo)
	}
}

// Shutdown() forcibly closes all open connections. This results in
// "503 Service Unavailable" responses to all outstanding requests.
func (ac *AsyncClient) Shutdown() {
	ac.lk.Lock()
	defer ac.lk.Unlock()
	ac.shut = true

	for s, r := range ac.hostmap {
		elm := r.connlist.Front()
		for elm != nil {
			scc := elm.Value.(*stampedClientConn)
			conn, _, _ := scc.Close()
			if conn != nil {
				conn.Close()
			}
			elm = elm.Next()
		}
		r.connlist.Init()
		ac.hostmap[s] = nil, false
	}
}
