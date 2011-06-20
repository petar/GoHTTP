// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"os"
	"testing"
	"github.com/petar/GoHTTP/http"
	"github.com/petar/GoHTTP/server"
)

type Service struct{}

type In struct {
	I0 int     "intv"
	F0 float32 "floatv"
	S0 string  "stringv"
	B0 bool    "boolv"
}

type Out struct {
	In In
}

func (s *Service) A(in *In, out *Out) os.Error {
	out.In = *in
	return nil
}

func TestAPI(t *testing.T) {
	srv, err := server.NewServerEasy("0.0.0.0:3232")
	if err != nil {
		t.Fatalf("starting server: %s", err)
	}
	api := NewAPI()
	err = api.Register(&Service{})
	if err != nil {
		t.Fatalf("Service register: %s", err)
	}
	srv.AddSub("/api/", api)
	for {
		q, err := srv.Read()
		if err != nil {
			t.Errorf("Problem: %s\n", err)
		}
		q.ContinueAndWrite(http.NewResponse404())
	}
}
