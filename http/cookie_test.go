// Copyright 2010 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"bytes"
	"reflect"
	"testing"
)


var writeSetCookiesTests = []struct {
	Cookies
	Raw string
}{
	{
		Cookies{"cookie-1": Cookie{Value: "v$1", MaxAge: -1}},
		"Set-Cookie: Cookie-1=v%241; \r\n",
	},
}

func TestWriteSetCookies(t *testing.T) {
	for i, tt := range writeSetCookiesTests {
		var w bytes.Buffer
		tt.Cookies.writeSetCookies(&w)
		seen := string(w.Bytes())
		if seen != tt.Raw {
			t.Errorf("Test %d, expecting:\n%s\nGot:\n%s\n", i, tt.Raw, seen)
			continue
		}
	}
}

var writeCookiesTests = []struct {
	Cookies
	Raw string
}{
	{
		Cookies{"cookie-1": Cookie{Value: "v$1", MaxAge: -1}},
		"Cookie: Cookie-1=v%241; \r\n",
	},
}

func TestWriteCookies(t *testing.T) {
	for i, tt := range writeCookiesTests {
		var w bytes.Buffer
		tt.Cookies.writeCookies(&w)
		seen := string(w.Bytes())
		if seen != tt.Raw {
			t.Errorf("Test %d, expecting:\n%s\nGot:\n%s\n", i, tt.Raw, seen)
			continue
		}
	}
}

var readSetCookiesTests = []struct {
	Header Header
	Cookies
}{
	{
		Header{"Set-Cookie": {"Cookie-1=v%241; "}},
		Cookies{"Cookie-1": Cookie{Value: "v$1", MaxAge: -1, Raw: "Cookie-1=v%241; "}},
	},
}

func TestReadSetCookies(t *testing.T) {
	for i, tt := range readSetCookiesTests {
		c := readSetCookies(tt.Header)
		if !reflect.DeepEqual(map[string]Cookie(*c), map[string]Cookie(tt.Cookies)) {
			t.Errorf("#%d readSetCookies: have\n%#v\nwant\n%#v\n", i, (*c), tt.Cookies)
			continue
		}
	}
}

var readCookiesTests = []struct {
	Header Header
	Cookies
}{
	{
		Header{"Cookie": {"Cookie-1=v%241; "}},
		Cookies{"Cookie-1": Cookie{Value: "v$1", MaxAge: -1, Raw: "Cookie-1=v%241; "}},
	},
}

func TestReadCookies(t *testing.T) {
	for i, tt := range readCookiesTests {
		c := readCookies(tt.Header)
		if !reflect.DeepEqual(map[string]Cookie(*c), map[string]Cookie(tt.Cookies)) {
			t.Errorf("#%d readCookies: have\n%#v\nwant\n%#v\n", i, (*c), tt.Cookies)
			continue
		}
	}
}
