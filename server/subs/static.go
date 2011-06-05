// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package subs

import (
	"path"
	"github.com/petar/GoHTTP/http"
	"github.com/petar/GoHTTP/server"
)

// StaticSub is a Sub that serves static files from a given directory.
// TODO: Intelligent caching
type StaticSub struct {
	staticPath string
}

func NewStaticSub(staticPath string) *StaticSub {
	return &StaticSub{staticPath}
}

func (ss *StaticSub) Serve(q *server.Query) {
	req := q.Req
	if req.Method != "GET" {
		q.ContinueAndWrite(http.NewResponse404(req))
		return
	}
	p := req.URL.Path
	if len(p) == 0 {
		p = "index.html"
	} else if p[0] == '/' {
		p = p[1:]
	}
	full := path.Join(ss.staticPath, p)
	resp, _ := http.NewResponseFile(req, full)
	q.ContinueAndWrite(resp)
}
