// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cache

import (
	"io/ioutil"
	"os"
	"sync"
)

// CachedFile is responsible for returning the contents of a single file.
// It remembers the contents in memory, and updates it as necessary.
type CachedFile struct {
	sync.Mutex
	fname string
	data  []byte
	mtime int64
}

func NewCachedFile(filename string) *CachedFile {
	return &CachedFile{fname: filename}
}

func (c *CachedFile) Get() (data []byte, err error) {
	c.Lock()
	defer c.Unlock()

	if c.data == nil {
		return c.readFile()
	}
	fi, err := os.Stat(c.fname)
	if err != nil {
		return nil, err
	}
	if fi.ModTime().UnixNano() > c.mtime {
		return c.readFile()
	}
	return c.data, nil
}

func (c *CachedFile) readFile() (data []byte, err error) {
	fi, err := os.Stat(c.fname)
	if err != nil {
		return nil, err
	}
	data, err = ioutil.ReadFile(c.fname)
	if err != nil {
		return nil, err
	}
	c.data = data
	c.mtime = fi.ModTime().UnixNano()

	return data, nil
}
