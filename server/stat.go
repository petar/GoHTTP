// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// Stats maintains server statistics and methods for
// querying into them.
type Stats struct {
	TimeStarted     int64  // Time server started
	RequestCount    uint64 // Number of request successfully received
	ResponseCount   uint64 // Number of responses successfully received
	ExpireConnCount uint64 // Number of connections, expired by the server
	AcceptConnCount uint64
	lk              sync.Mutex
}

func (s *Stats) Init() {
	s.TimeStarted = time.Nanoseconds()
}

func (s *Stats) IncRequest() {
	s.lk.Lock()
	defer s.lk.Unlock()
	s.RequestCount++
}

func (s *Stats) IncResponse() {
	s.lk.Lock()
	defer s.lk.Unlock()
	s.ResponseCount++
}

func (s *Stats) IncExpireConn() {
	s.lk.Lock()
	defer s.lk.Unlock()
	s.ExpireConnCount++
}

func (s *Stats) IncAcceptConn() {
	s.lk.Lock()
	defer s.lk.Unlock()
	s.AcceptConnCount++
}

func (s *Stats) SummaryLine() string {
	s.lk.Lock()
	defer s.lk.Unlock()
	return fmt.Sprintf("Running %d mins, %d accept, %d expire, %d req, %d resp; %d goroutine\n",
		(time.Nanoseconds()-s.TimeStarted)/(60*1e9),
		s.AcceptConnCount, s.ExpireConnCount, s.RequestCount, s.ResponseCount,
		runtime.Goroutines())
}
