// Copyright 2017 Fabian Kohn. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package profiler defines and manages the basic profiling commands and the
// web frontend.
package profiler

import "net/http"

// defaultAddr is the default address (ip:port) to listen on
const defaultAddr = "127.0.0.1:6060"

// defaultErrorHandler is the default error handler for the web frontend (which
// does nothing)
var defaultErrorHandler = func(error) {}

// Profiler represents a profiler interface instance
type Profiler struct {
	server            *http.Server                    // web server
	middleware        func(http.Handler) http.Handler // additional middleware
	certFile, keyFile string                          // TLS key/certificate files
	htmlTemplate      string                          // HTML template for index page
	errorHandler      func(error)                     // error handler function for critical frontend issues
}

// New creates and returns a new debugger instance
func New(options ...func(*Profiler)) *Profiler {

	// Initialize new debugger (default values can be overridden by functional options)
	p := &Profiler{
		server: &http.Server{
			Addr: defaultAddr,
		},
		htmlTemplate: defaultHTMLTemplate,
		errorHandler: defaultErrorHandler,
	}

	// Execute functional options (if any), see options.go for implementation
	for _, option := range options {
		option(p)
	}

	// Set / register handlers for debugger interface
	p.registerHandlers()

	return p
}

// Run starts the debugger (wrapping an optional error handler, which will perform
// the specified action if there is any issue inside the goroutine). TLS is used
// in case key and certificate are provided
func (p *Profiler) Run() {
	go func() {
		if p.certFile != "" && p.keyFile != "" {
			p.errorHandler(p.server.ListenAndServeTLS(p.certFile, p.keyFile))
		} else {
			p.errorHandler(p.server.ListenAndServe())
		}
	}()
}

func (p *Profiler) registerHandlers() {

	// Initialize a new muxer
	muxer := http.NewServeMux()

	// Register the different methods on their respective URIs
	muxer.HandleFunc("/", http.HandlerFunc(p.Index))
	muxer.HandleFunc("/profile", http.HandlerFunc(p.Profile))
	muxer.HandleFunc("/cmdline", http.HandlerFunc(p.Cmdline))
	muxer.HandleFunc("/symbol", http.HandlerFunc(p.Symbol))
	muxer.HandleFunc("/trace", http.HandlerFunc(p.Trace))

	// Set custom middleware (if provided)
	if p.middleware == nil {
		p.server.Handler = muxer
	} else {
		p.server.Handler = p.middleware(muxer)
	}
}

// defaultHTMLTemplate is the standard HTML index page template
const defaultHTMLTemplate = `<html>
<head>
<title>Go Profiling interface</title>
</head>
<body>
<b>Go Profiling interface</b><br>
<br>
<b>CPU profile:</b><br>
<table>
<tr><td align=left>plain text<td>(<a href="profile?seconds=5">5s</a> <a href="profile?seconds=15">15s</a> <a href="profile?seconds=30">30s</a> <a href="profile?seconds=60">1min</a>)
<tr><td align=left>binary<td>(<a href="profile?seconds=5&binary=true">5s</a> <a href="profile?seconds=15&binary=true">15s</a> <a href="profile?seconds=30&binary=true">30s</a> <a href="profile?seconds=60&binary=true">1min</a>)
</table>
<br>
<b>Execution trace:</b><br>
<table>
<tr><td align=left>binary<td>(<a href="trace?seconds=0.1">0.1s</a> <a href="trace?seconds=0.5">0.5s</a> <a href="trace?seconds=1.0">1.0s</a>)
</table>
<br>
<b>Available default profiles:</b><br>
<table>
{{range .}}
<tr><td align=left><a href="{{.Name}}?debug=1">{{.Name}}</a><td> ({{.Count}})
{{end}}
<tr><td align=left><a href="goroutine?debug=2">stack dump</a>
</table>
</body>
</html>
`
