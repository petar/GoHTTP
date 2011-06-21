// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"fmt"
	"os"
	"testing"
	"github.com/petar/GoHTTP/http"
	"github.com/petar/GoHTTP/server"
)

type Service struct{}

func (s *Service) FS(args *ShortArgs, ret *ShortRet) os.Error {
	fmt.Printf("Args--\n%#v\n--\n", args)
	ret.Value = make(map[string]string)
	ret.Value["key0"] = "val0"
	return nil
}

func (s *Service) FL(args *LongArgs, ret *LongRet) os.Error {
	fmt.Printf("Args--\n%#v\n--\n", args)
	ret.Value = make(map[string][]string)
	ret.Value["key0"] = []string{"val0"}
	return nil
}

func (s *Service) FSC(args *ShortCookieArgs, ret *ShortSetCookieRet) os.Error {
	fmt.Printf("Args--\n%#v\n--\n", args)
	ret.Value = make(map[string]string)
	ret.Value["key0"] = "val0"
	return nil
}

func (s *Service) FLC(args *LongCookieArgs, ret *LongSetCookieRet) os.Error {
	fmt.Printf("Args--\n%#v\n--\n", args)
	if len(args.Cookies) > 0 {
		fmt.Printf("Cookie0--\n%#v\n--\n", args.Cookies[0])
	}
	ret.Value = make(map[string][]string)
	ret.Value["key0"] = []string{"val0"}
	ret.SetCookies = make([]*http.Cookie, 1)
	ret.SetCookies[0] = &http.Cookie{
		Name:  "TestCookie",
		Value: "TestCookieValue",
	}
	return nil
}

func TestRPC(t *testing.T) {
	srv, err := server.NewServerEasy("0.0.0.0:3300")
	if err != nil {
		t.Fatalf("starting server: %s", err)
	}
	rpcs := NewRPC()
	err = rpcs.RegisterName("s", &Service{})
	if err != nil {
		t.Fatalf("service register: %s", err)
	}
	srv.AddSub("/api/", rpcs)
	srv.Launch()
}
