// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"os"
	"github.com/petar/GoHTTP/http"
)

// An Extension is a piece of server-side logic that can perform
// a few functions. It can claim a URL-subspace and it can attach
// itself to the header processing chains for incoming requests
// and outgoing respones.
type Extension interface {
	ReadRequest(req *http.Request, ext map[string]interface{}) os.Error
	WriteResponse(resp *http.Response, ext map[string]interface{}) os.Error
}

type ExtensionConfig struct {
	Name             string
	RequestSubspace  string
	ResponseSubspace string
}
