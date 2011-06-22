// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"os"
)

// Convenience RPC arguments value structures

type LongCookieArgs struct {
	Cookies []*http.Cookie
	Value   map[string][]string
}

type ShortCookieArgs struct {
	Cookies []*http.Cookie
	Value   map[string]string
}

type LongArgs struct {
	Value map[string][]string
}

type ShortArgs struct {
	Value map[string]string
}

type CookieArgs struct {
	Cookies []*http.Cookie
}

type NoArgs struct {}

// Convenience RPC return values structures

type LongSetCookieRet struct {
	SetCookies []*http.Cookie
	Value      map[string][]string
}

type ShortSetCookieRet struct {
	SetCookies []*http.Cookie
	Value      map[string]string
}

type LongRet struct {
	Value map[string][]string
}

type ShortRet struct {
	Value map[string]string
}

type SetCookieRet struct {
	SetCookies []*http.Cookie
}

type NoRet struct {}

var (
	ErrArg = os.NewError("bad or missing RPC argument")
)

func GetBool(arg *ShortArgs, key string) (bool, os.Error) {
	if arg.Value == nil {
		return false, ErrArg
	}
	v, ok := arg.Value[key]
	if !ok {
		return false, ErrArg
	}
	if v == "0" {
		return false, nil
	}
	if v == "1" {
		return true, nil
	}
	return false, ErrArg
}

func SetBool(r *ShortArgs, key string, value bool) {
	if r.Value == nil {
		r.Value = make(map[string]string)
	}
	s := "0"
	if value {
		s = "1"
	}
	r.Value[key] = s
}
