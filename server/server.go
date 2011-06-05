// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	//"fmt"
	"container/list"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
	"github.com/petar/GoHTTP/http"
	"github.com/petar/GoHTTP/util"
)

// Server automates the reception of incoming HTTP connections
// at a given net.Listener. Server accepts new connections and
// manages each one with an ServerConn object. Server also
// makes sure that a pre-specified limit of active connections (i.e.
// file descriptors) is not exceeded.
type Server struct {
	sync.Mutex // protects listen and conns

	// Real-time state
	listen net.Listener
	conns  map[*StampedServerConn]int
	qch    chan *Query
	fdl    util.FDLimiter
	subs   []*subcfg
	exts   []*extcfg

	config Config // Server configuration
	stats  Stats  // Real-time statistics
}

// NewServer creates a new Server which listens for connections on l.
// New connections are automatically managed by ServerConn objects with
// timout set to tmo nanoseconds. The Server object ensures that at no
// time more than fdlim file descriptors are allocated to incoming connections.
func NewServer(l net.Listener, config Config, fdlim int) *Server {
	if config.Timeout < 2 {
		panic("timeout too small")
	}
	// TODO(petar): Perhaps a better design passes the FDLimiter as a parameter
	srv := &Server{
		config: config,
		listen: l,
		conns:  make(map[*StampedServerConn]int),
		qch:    make(chan *Query),
	}
	srv.fdl.Init(fdlim)
	srv.stats.Init()
	go srv.acceptLoop()
	go srv.expireLoop()
	return srv
}

func NewServerEasy(addr string) (*Server, os.Error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return NewServer(l, Config{5e9}, 200), nil
}

func (srv *Server) GetFDLimiter() *util.FDLimiter { return &srv.fdl }

func (srv *Server) expireLoop() {
	for i := 0; ; i++ {
		srv.Lock()
		if srv.listen == nil {
			srv.Unlock()
			return
		}
		now := time.Nanoseconds()
		kills := list.New()
		for ssc, _ := range srv.conns {
			if now-ssc.GetStamp() >= srv.config.Timeout {
				kills.PushBack(ssc)
				srv.stats.IncExpireConn()
			}
		}
		srv.Unlock()
		elm := kills.Front()
		for elm != nil {
			ssc := elm.Value.(*StampedServerConn)
			srv.bury(ssc)
			elm = elm.Next()
		}
		kills.Init()
		kills = nil
		time.Sleep(srv.config.Timeout)
		if i%4 == 0 {
			log.Println(srv.stats.SummaryLine())
		}
	}
}

func (srv *Server) acceptLoop() {
	for {
		srv.Lock()
		l := srv.listen
		srv.Unlock()
		if l == nil {
			return
		}
		srv.fdl.Lock()
		c, err := l.Accept()
		if err != nil {
			if c != nil {
				c.Close()
			}
			srv.fdl.Unlock()
			srv.qch <- newQueryErr(err)
			return
		}
		srv.stats.IncAcceptConn()
		c.(*net.TCPConn).SetKeepAlive(true)
		err = c.SetReadTimeout(srv.config.Timeout)
		if err != nil {
			log.Printf("Set read timeout: %s\n", err)
			c.Close()
			srv.fdl.Unlock()
			srv.qch <- newQueryErr(err)
			return
		}
		err = c.SetWriteTimeout(srv.config.Timeout)
		if err != nil {
			log.Printf("Set write timeout: %s\n", err)
			c.Close()
			srv.fdl.Unlock()
			srv.qch <- newQueryErr(err)
			return
		}
		c = util.NewRunOnCloseConn(c, func() { srv.fdl.Unlock() })
		ssc := NewStampedServerConn(c, nil)
		srv.register(ssc)
		go srv.read(ssc)
	}
}

// Read() waits until a new request is received. The request is
// returned in the form of a Query object. A returned error
// indicates that the Server cannot accept new connections,
// and the user us expected to call Shutdown(), perhaps after serving
// outstanding queries.
func (srv *Server) Read() (query *Query, err os.Error) {
	// TODO: This loop processes requests in sequence. And does not process a new one
	// until the old one has processed in process(). Need to parallelize this.
	for {
		q, ok := <-srv.qch
		srv.Lock()
		if !ok {
			srv.Unlock()
			return nil, os.EBADF
		}
		srv.Unlock()
		if err = q.getError(); err != nil {
			return nil, err
		}
		q = srv.process(q)
		if q != nil {
			return q, nil
		}
	}
	panic("unreach")
}

func (srv *Server) Launch() os.Error {
	for {
		q, err := srv.Read()
		if err != nil {
			return err
		}
		q.ContinueAndWrite(http.NewResponse404(q.Req))
	}
	panic("unreach")
}

func (srv *Server) AddSub(url string, sub Sub) {
	srv.Lock()
	defer srv.Unlock()
	srv.subs = append(srv.subs, &subcfg{url, sub})
}

func (srv *Server) AddExt(name, url string, ext Extension) {
	srv.Lock()
	defer srv.Unlock()
	srv.exts = append(srv.exts, &extcfg{name, url, ext})
}

func (srv *Server) subIter() chan *subcfg {
	srv.Lock()
	defer srv.Unlock()

	ss := make([]*subcfg, len(srv.subs))
	copy(ss, srv.subs)
	ch := make(chan *subcfg)
	go func() {
		for _, s := range ss {
			ch <- s
		}
		close(ch)
	}()
	return ch
}

func (srv *Server) extIter() chan *extcfg {
	srv.Lock()
	defer srv.Unlock()

	ee := make([]*extcfg, len(srv.exts))
	copy(ee, srv.exts)
	ch := make(chan *extcfg)
	go func() {
		for _, e := range ee {
			ch <- e
		}
		close(ch)
	}()
	return ch
}

func (srv *Server) extRevIter() chan *extcfg {
	srv.Lock()
	defer srv.Unlock()

	ee := make([]*extcfg, len(srv.exts))
	copy(ee, srv.exts)
	ch := make(chan *extcfg)
	go func() {
		for i := 0; i < len(ee); i++ {
			ch <- ee[len(ee)-i-1]
		}
		close(ch)
	}()
	return ch
}

func (srv *Server) process(q *Query) *Query {

	// Apply extensions
	p := q.origPath

	q.Ext = make(map[string]interface{})
	extch := srv.extIter()
	for ec, ok := <-extch; ok; ec, ok = <-extch {
		if strings.HasPrefix(p, ec.SubURL) {
			if err := ec.Ext.ReadRequest(q.Req, q.Ext); err != nil {
				return nil
			}
		}
	}

	// Serve using a sub?
	p = q.Req.URL.Path
	subch := srv.subIter()
	for sc, ok := <-subch; ok; sc, ok = <-subch {
		if strings.HasPrefix(p, sc.SubURL) {
			q.Req.URL.Path = p[len(sc.SubURL):]
			sc.Sub.Serve(q)
			return nil
		}
	}

	return q
}

func (srv *Server) read(ssc *StampedServerConn) {
	for {
		req, err := ssc.Read()
		perr, ok := err.(*os.PathError)
		if ok && perr.Error == os.EAGAIN {
			log.Printf("Request Read path error: Op=%s, Path=%s, Error=%s\n", perr.Op, perr.Path, perr.Error)
			srv.bury(ssc)
			return
		}
		if err != nil {
			// TODO(petar): Technically, a read side error should not terminate
			// the ServerConn if there are outstanding requests to be answered,
			// since the write side might still be healthy. But this is
			// virtually never the case with TCP, so we currently go for simplicity
			// and just close the connection.

			// NOTE(petar): 'tcp read ... resource temporarily unavailable' errors 
			// received here, I think, correspond to when the remote side has closed
			// the connection. This is OK.
			srv.bury(ssc)
			return
		}
		srv.qch <- &Query{
			Req:      req,
			srv:      srv,
			ssc:      ssc,
			origPath: req.URL.Path,
		}
		srv.stats.IncRequest()
		return
	}
}

func (srv *Server) register(ssc *StampedServerConn) {
	srv.Lock()
	defer srv.Unlock()
	if _, present := srv.conns[ssc]; present {
		panic("register twice")
	}
	srv.conns[ssc] = 1
}

func (srv *Server) unregister(ssc *StampedServerConn) {
	srv.Lock()
	defer srv.Unlock()
	srv.conns[ssc] = 0, false
}

func (srv *Server) bury(ssc *StampedServerConn) {
	srv.unregister(ssc)
	ssc.Close()
}

// Shutdown closes the Server by closing the underlying
// net.Listener object. The user should not use any Server
// or Query methods after a call to Shutdown.
func (srv *Server) Shutdown() (err os.Error) {
	// First, close the listener
	srv.Lock()
	var l net.Listener
	l, srv.listen = srv.listen, nil
	close(srv.qch)
	srv.Unlock()
	if l != nil {
		err = l.Close()
	}
	// Then, force-close all open connections
	srv.Lock()
	for ssc, _ := range srv.conns {
		ssc.Close()
		srv.conns[ssc] = 0, false
	}
	srv.Unlock()
	return
}
