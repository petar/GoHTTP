// Copyright 2010 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	/*
		"bufio"
		"bytes"
		"fmt"
		"io"
		"reflect"
	*/
	"os"
	"testing"
)

/* extractSetCookie test */

type extractSetCookieTest struct {
	Raw    string
	Parsed Cookies
}

var extractSetCookieTests = []extractSetCookieTest{}

func TestOK(t *testing.T) {
	kk := &Cookies{
		"aha":  Cookie{Value: "toto", HttpOnly: true},
		"bibo": Cookie{Value: "doce", HttpOnly: true},
	}
	if err := kk.writeSetCookies(os.Stdout); err != nil {
		t.Errorf("** %s\n", err)
	}
}

/*
func TestReadResponse(t *testing.T) {
	for i := range respTests {
		tt := &respTests[i]
		var braw bytes.Buffer
		braw.WriteString(tt.Raw)
		resp, err := ReadResponse(bufio.NewReader(&braw), tt.Resp.RequestMethod)
		if err != nil {
			t.Errorf("#%d: %s", i, err)
			continue
		}
		rbody := resp.Body
		resp.Body = nil
		diff(t, fmt.Sprintf("#%d Response", i), resp, &tt.Resp)
		var bout bytes.Buffer
		if rbody != nil {
			io.Copy(&bout, rbody)
			rbody.Close()
		}
		body := bout.String()
		if body != tt.Body {
			t.Errorf("#%d: Body = %q want %q", i, body, tt.Body)
		}
	}
}

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
