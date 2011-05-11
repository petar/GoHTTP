// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"os"
	"template"
)

// CachedTemplate represents a template.Template object that comes from a file.
// A get-method returns the template.Template object, while transparently ensuring
// that the returned object corresponds to the newest version of the file.
// NOTE: CachedTemplate is not re-entrant because template.Template is not anyway.
type CachedTemplate struct {
	fname  string
	fmap   template.FormatterMap
	templ  *template.Template
	mtime  int64
}

func NewCachedTemplate(filename string, fmap template.FormatterMap) *CachedTemplate {
	return &CachedTemplate{ fname: filename, fmap: fmap }
}

func (c *CachedTemplate) Get() (templ *template.Template, err os.Error) {
	if c.templ == nil {
		return c.readFile()
	}
	fi, err := os.Stat(c.fname)
	if err != nil {
		return nil, err
	}
	if fi.Mtime_ns > c.mtime {
		return c.readFile()
	}
	return c.templ, nil
}

// readFile creates a new template object directly from the corresponding file.
func (c *CachedTemplate) readFile() (templ *template.Template, err os.Error) {
	fi, err := os.Stat(c.fname)
	if err != nil {
		return nil, err
	}
	templ, err = template.ParseFile(c.fname, c.fmap)
	if err != nil {
		return nil, err
	}
	c.templ = templ
	c.mtime = fi.Mtime_ns
	return templ, nil
}
