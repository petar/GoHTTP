// Copyright 2010 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"bytes"
	"net/textproto"
	"reflect"
	"testing"
)

/* writeSetCookies test */

type writeSetCookiesTest struct {
	Cookies
	Raw string
}

var writeSetCookiesTests = []writeSetCookiesTest{
	{
		Cookies{"cookie-1": Cookie{ Value: "v$1", MaxAge: -1 }}, 
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

/* writeCookies test */

type writeCookiesTest struct {
	Cookies
	Raw string
}

var writeCookiesTests = []writeCookiesTest{
	{
		Cookies{"cookie-1": Cookie{ Value: "v$1", MaxAge: -1 }}, 
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

/* readSetCookies test */

type readSetCookiesTest struct {
	Header textproto.MIMEHeader
	Cookies
}

var readSetCookiesTests = []readSetCookiesTest{
	{
		textproto.MIMEHeader{"Set-Cookie": {"Cookie-1=v%241; "}},
		Cookies{"Cookie-1": Cookie{ Value: "v$1", MaxAge: -1, Raw: "Cookie-1=v%241" }}, 
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

/*
func diff(t *testing.T, prefix string, have, want interface{}) {
	hv := reflect.NewValue(have).(*reflect.PtrValue).Elem().(*reflect.StructValue)
	wv := reflect.NewValue(want).(*reflect.PtrValue).Elem().(*reflect.StructValue)
	if hv.Type() != wv.Type() {
		t.Errorf("%s: type mismatch %v vs %v", prefix, hv.Type(), wv.Type())
	}
	for i := 0; i < hv.NumField(); i++ {
		hf := hv.Field(i).Interface()
		wf := wv.Field(i).Interface()
		if !reflect.DeepEqual(hf, wf) {
			t.Errorf("%s: %s = %v want %v", prefix, hv.Type().(*reflect.StructType).Field(i).Name, hf, wf)
		}
	}
}
*/
