## Description

GoHTTP is an early draft, more like a sketch, of a high-performance Go
HTTP server framework. It uses the low-level _"net/http/httputil"_ standard
package, which provides access to advanced features like pipelining, and is
not yet utilized within the standard Go _"net/http"_ infrastructure.

The main point of interest is that decent HTTP proxies must suppport such
features, and therefore this draft I am current reviving seems like the only
route to implementing a workable web HTTP proxy in Go.

## Features

* Core web server infrastructure with out-of-the-box keepalive and pipelining
	* Support for limiting the number of file descriptors in use
* A "query" abstraction to handling incoming requests which is more convenient than that of Go's the HTTP package
* A system of "sub-servers" that allows modular handling of different URL sub-paths
* A system of "extensions" which allows modular pre- and post-processing of HTTP header objects like cookies
* A sub-server module for serving static files with basic in-memory file caching

## About

GoHTTP is maintained by [Petar Maymounkov](http://maymounkov.org/). 
