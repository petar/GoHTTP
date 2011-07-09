// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cache

import (
	"os"
	"mime"
	"path"
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

func (c *Cache) Get(filename string) (content []byte, mimetype string, err os.Error) {
	c.Lock()
	f, ok := c.files[filename]
	if !ok {
		f = NewCachedFile(filename)
		c.files[filename] = f
	}
	c.Unlock()
	content, err = f.Get()
	if err == nil {
		mimetype = mime.TypeByExtension(path.Ext(filename))
	}
	return content, mimetype, err
}
