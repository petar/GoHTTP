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
	MaxReqRespTime  uint64 // Duration of longest request-response cycle
	lk              sync.Mutex
}

func (s *Stats) Init() {
	s.TimeStarted = time.Nanoseconds()
}

func (s *Stats) AddReqRespTime(d int64) {
	s.lk.Lock()
	defer s.lk.Unlock()
	if uint64(d) > s.MaxReqRespTime {
		s.MaxReqRespTime = uint64(d)
	}
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
	return fmt.Sprintf("Running %d mins, %d accept, %d expire, %d req, %d resp; MaxReqRespTime: %dms; %d goroutine",
		(time.Nanoseconds()-s.TimeStarted)/(60*1e9),
		s.AcceptConnCount, s.ExpireConnCount, s.RequestCount, s.ResponseCount,
		s.MaxReqRespTime/1e6,
		runtime.Goroutines())
}
