// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

type subserver struct {
	SubURL    string
	Subserver Subserver
}

type Subserver interface {
	Serve(q *Query)
}
