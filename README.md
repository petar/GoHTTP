## Description

GoHTTP is a growing web server infrastructure for Go. It includes
things like serving static pages, writing REST APIs and more.
(See the section on features below.)

GoHTTP uses its own bleeding edge version of Google Go's HTTP package. 
Differences between GoHTTP's version of the HTTP package are eventually 
submitted to the main Google Go library.

The packages in this project are used "in production" on a few
sites that I run, like e.g. 

* [Population algorithms](http://popalg.org)
* [The Tonika, aka 5ttt.org, blog](http://blog.5ttt.org)

## Features

* Core web server infrastructure with out-of-the-box keepalive and pipelining
	* Support for limiting the number of file descriptors in use
* A "query" abstraction to handling incoming requests which is more convenient than that of Go's the HTTP package
* A system of "sub-servers" that allows modular handling of different URL sub-paths
* A system of "extensions" which allows modular pre- and post-processing of HTTP header objects like cookies
* A sub-server module for serving static files with basic in-memory file caching

## Installation

GoHTTP entails multiple packages. With a working installation of Go, 
they can be installed like so

	goinstall github.com/petar/GoHTTP/http
	goinstall github.com/petar/GoHTTP/cache
	goinstall github.com/petar/GoHTTP/util
	goinstall github.com/petar/GoHTTP/template
	goinstall github.com/petar/GoHTTP/server
		and so on ... 

## About

GoHTTP is maintained by [Petar Maymounkov](http://pdos.csail.mit.edu/~petar/). 
