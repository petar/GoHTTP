// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exts

import (
	"path"
	"github.com/petar/GoHTTP/http"
	"github.com/petar/GoHTTP/server"
)

type Session struct {
}

func NewSession() *Session {
}

func (s *Session) ReadRequest(req *http.Request, ext map[string]interface{}) os.Error {
}

func (s *Session) WriteResponse(resp *http.Response, ext map[string]interface{}) os.Error {
}
