// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"io"
	"os"
	"strings"
)

type Cookie struct {
	Value    string
	Path     string
	Domain   string
	Comment  string
	Version  string
	Expires  int64
	MaxAge   int64
	Secure   bool
	HttpOnly bool
	Raw      string
}

type Cookies map[string]Cookie

func ExtractSetCookies(h map[string][]string) *Cookies {
	kk := new(Cookies)
	sk, ok := h["Set-Cookie"]
	if !ok {
		return kk
	}
	for _, ktext := range sk {
		parts := strings.Split(ktext, ";", -1)
		if len(parts) == 0 {
			continue
		}
		// XXX encodings??
		nv := strings.Split(strings.TrimSpace(parts[0]), "=", 2)		// Name=Value
		if len(nv) != 2 {
			continue
		}
		c := Cookie{ Value: nv[1] }
		for i := 1; i < len(parts); i++ {
			av := strings.Split(strings.TrimSpace(parts[i]), "=", 2)	// Attribute=Value
			if len(av) == 1 {
				switch strings.ToLower(av[0]) {
				case "secure":
					c.Secure = true
				case "httponly":
					c.HttpOnly = true
				}
			} else if len(av) == 2 {
				switch strings.ToLower(av[0]) {
				case "comment":
					c.Comment = av[1]
				case "domain":
					c.Domain = av[1]
				case "max-age":
					// ??
				case "expires":
					// ??
				case "path":
					c.Path = av[1]
					// ??
				case "secure":
					c.Secure = true
				case "version":
					c.Version = av[1]
				case "httponly":
					c.HttpOnly = true
				}
			}
		}
		(*kk)[nv[0]] = c
	}
	h["Set-Cookie"] = nil, false
	return kk
}

func (ck *Cookie) Write(w io.Writer) os.Error {
	return nil
}
