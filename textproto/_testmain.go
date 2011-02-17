package main

import "net/textproto"
import "testing"
import __regexp__ "regexp"

var tests = []testing.InternalTest{
	{"textproto.TestCanonicalHeaderKey", textproto.TestCanonicalHeaderKey},
	{"textproto.TestReadLine", textproto.TestReadLine},
	{"textproto.TestReadContinuedLine", textproto.TestReadContinuedLine},
	{"textproto.TestReadCodeLine", textproto.TestReadCodeLine},
	{"textproto.TestReadDotLines", textproto.TestReadDotLines},
	{"textproto.TestReadDotBytes", textproto.TestReadDotBytes},
	{"textproto.TestReadMIMEHeader", textproto.TestReadMIMEHeader},
	{"textproto.TestPrintfLine", textproto.TestPrintfLine},
	{"textproto.TestDotWriter", textproto.TestDotWriter},
}
var benchmarks = []testing.InternalBenchmark{ //
}

func main() {
	testing.Main(__regexp__.MatchString, tests)
	testing.RunBenchmarks(__regexp__.MatchString, benchmarks)
}
