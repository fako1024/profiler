// Copyright 2017 Fabian Kohn. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package profiler defines and manages the basic profiling commands and the
// web frontend.
package profiler

import "net/http"

// WithServer sets a custom http.Server to serve the debugger
func WithServer(server *http.Server) func(*Profiler) {
	return func(p *Profiler) {
		p.server = server
	}
}

// WithAddr sets a custom endpoint (ip:port) for the debugger
func WithAddr(addr string) func(*Profiler) {
	return func(p *Profiler) {
		p.server.Addr = addr
	}
}

// WithKeyPair sets private / public key files used for the debugger web interface
func WithKeyPair(certFile, keyFile string) func(*Profiler) {
	return func(p *Profiler) {
		p.certFile, p.keyFile = certFile, keyFile
	}
}

// WithMiddleware defines a custom middleware which will be wrapped around the
// basic web interface handler (e.g. authentication scheme and/or logger)
// NOTE: Arbitrarily nested / chained handlers can easily be achieved by providing
// an already chained middleware handler to this function.
func WithMiddleware(middleware func(http.Handler) http.Handler) func(*Profiler) {
	return func(p *Profiler) {
		p.middleware = middleware
	}
}

// WithHTMLTemplate sets a custom HTML template for the debugger index page
func WithHTMLTemplate(template string) func(*Profiler) {
	return func(p *Profiler) {
		p.htmlTemplate = template
	}
}

// WithErrorHandler sets an error handler function for the http.ListenAndServe[TLS]
// call
func WithErrorHandler(handlerFunc func(error)) func(*Profiler) {
	return func(p *Profiler) {
		p.errorHandler = handlerFunc
	}
}
