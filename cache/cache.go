// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cache

import (
	"os"
	"sync"
)

type Cache struct {
	sync.Mutex
	files map[string]*CachedFile
}

func NewCache() *Cache {
	return &Cache{
		files: make(map[string]*CachedFile),
	}
}

func (c *Cache) Get(filename string) ([]byte, os.Error) {
	c.Lock()
	f, ok := c.files[filename]
	if !ok {
		f = NewCachedFile(filename)
		c.files[filename] = f
	}
	c.Unlock()
	return f.Get()
}
