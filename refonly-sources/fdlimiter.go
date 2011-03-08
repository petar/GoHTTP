// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"os"
	"sync"
	"time"
)

// FDLimiter helps keep track of the number of file descriptors in use.
type FDLimiter struct {
	limit int
	count int
	lk    sync.Mutex
	ch    chan int
	nfych chan<- int
}

// Init initializes (or resets) an FDLimiter object.
func (fdl *FDLimiter) Init(fdlim int) {
	fdl.lk.Lock()
	if fdlim <= 0 {
		panic("FDLimiter, bad limit")
	}
	fdl.limit = fdlim
	fdl.count = 0
	fdl.ch = make(chan int)
	fdl.lk.Unlock()
}

// SetNotifyChan instructs the FDLimiter to send the current
// number of utilized file descriptors every time that number changes.
// Calling this method with a nil argument, removes the notify channel.
func (fdl *FDLimiter) SetNotifyChan(c chan<- int) {
	fdl.lk.Lock()
	fdl.nfych = c
	fdl.lk.Unlock()
}

func (fdl *FDLimiter) notify() {
	if fdl.nfych != nil {
		fdl.nfych <- fdl.count
	}
}

func (fdl *FDLimiter) LockCount() int {
	fdl.lk.Lock()
	defer fdl.lk.Unlock()
	return fdl.count
}

func (fdl *FDLimiter) Limit() int { return fdl.limit }

// Lock blocks until it can allocate one fd without violating the limit.
func (fdl *FDLimiter) Lock() {
	for {
		fdl.lk.Lock()
		if fdl.count < fdl.limit {
			fdl.count++
			fdl.notify()
			fdl.lk.Unlock()
			return
		}
		fdl.lk.Unlock()
		<-fdl.ch
	}
	panic("FDLimiter, unreachable")
}

// LockOrTimeout proceeds as Lock, except that it returns an os.EAGAIN
// error, if a lock cannot be obtained within ns nanoseconds.
func (fdl *FDLimiter) LockOrTimeout(ns int64) os.Error {
	waitsofar := int64(0)
	for {
		// Try to get an fd
		fdl.lk.Lock()
		if fdl.count < fdl.limit {
			fdl.count++
			fdl.notify()
			fdl.lk.Unlock()
			return nil
		}
		fdl.lk.Unlock()

		// Or, wait for an fd or timeout
		if waitsofar >= ns {
			return os.EAGAIN
		}
		t0 := time.Nanoseconds()
		alrm := alarmOnce(ns - waitsofar)
		select {
		case <-alrm:
		case <-fdl.ch:
		}
		waitsofar += time.Nanoseconds() - t0
	}
	panic("FDLimiter, unreachable")
}

func (fdl *FDLimiter) LockOrChan(ch <-chan interface{}) (msg interface{}, err os.Error) {
	for {
		fdl.lk.Lock()
		if fdl.count < fdl.limit {
			fdl.count++
			fdl.notify()
			fdl.lk.Unlock()
			return nil, nil
		}
		fdl.lk.Unlock()

		select {
		case msg = <-ch:
			return msg, os.EAGAIN
		case <-fdl.ch:
		}
	}
	panic("FDLimiter, unreachable")
}

// Call Unlock to indicate that a file descriptor has been released.
func (fdl *FDLimiter) Unlock() {
	fdl.lk.Lock()
	if fdl.count <= 0 {
		panic("FDLimiter")
	}
	fdl.count--
	fdl.notify()
	if fdl.count == fdl.limit-1 {
		fdl.ch <- 1
	}
	fdl.lk.Unlock()
}

// alarmOnce sends "1" to the returned chan after ns nanoseconds
func alarmOnce(ns int64) <-chan int {
	backchan := make(chan int)
	go func() {
		time.Sleep(ns)
		backchan <- 1
		close(backchan)
	}()
	return backchan
}
