// Copyright 2017 Fabian Kohn. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package profiler defines and manages the basic profiling commands and the
// web frontend.
package profiler

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"strings"
	"time"

	"github.com/fako1024/profiler/internal/fetch"
	"github.com/fako1024/profiler/internal/profile"
	"github.com/fako1024/profiler/internal/report"
	"github.com/fako1024/profiler/internal/symbolz"
)

// Handler returns an HTTP handler that serves the named profile.
func Handler(name string) http.Handler {
	return handler(name)
}

type handler string

func (name handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	debug, _ := strconv.Atoi(r.FormValue("debug"))
	p := pprof.Lookup(string(name))
	if p == nil {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Unknown profile: %s\n", name)
		return
	}
	gc, _ := strconv.Atoi(r.FormValue("gc"))
	if name == "heap" && gc > 0 {
		runtime.GC()
	}
	p.WriteTo(w, debug)
	return
}

// Index responds with the pprof-formatted profile named by the request.
// For example, "/heap" serves the "heap" profile.
// Index responds to a request for "/" with an HTML page
// listing the available profiles.
func (p *Profiler) Index(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != "/" {
		handler(strings.TrimPrefix(r.URL.Path, "/")).ServeHTTP(w, r)
		return
	}

	profiles := pprof.Profiles()
	indexTmpl := template.Must(template.New("index").Parse(p.htmlTemplate))

	if err := indexTmpl.Execute(w, profiles); err != nil {
		log.Print(err)
	}
}

// Cmdline responds with the running program's
// command line, with arguments separated by NUL bytes.
// The package initialization registers it as /cmdline.
func (p *Profiler) Cmdline(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, strings.Join(os.Args, "\x00"))
}

// Profile responds with the pprof-formatted cpu profile.
// The package initialization registers it as /profile.
func (p *Profiler) Profile(w http.ResponseWriter, r *http.Request) {
	sec, _ := strconv.ParseInt(r.FormValue("seconds"), 10, 64)
	if sec == 0 {
		sec = 30
	}

	binary, _ := strconv.ParseBool(r.FormValue("binary"))
	cum, _ := strconv.ParseBool(r.FormValue("cum"))

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	var buf bytes.Buffer

	if err := pprof.StartCPUProfile(&buf); err != nil {
		// StartCPUProfile failed, so no writes yet.
		// Enforce header to text content and send error code.
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Could not enable CPU profiling: %s\n", err)
		return
	}
	sleep(w, time.Duration(sec)*time.Second)
	pprof.StopCPUProfile()

	// If plain text profile was requested, process the collected profile data,
	// otherwise send the binary data in the buffer
	if binary {

		// Set binary content type and send the data
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(buf.Bytes())
	} else {

		// Parse profile
		prof, err := profile.Parse(&buf)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to parse profile: %s\n", err)
			return
		}

		// Symbolize profile using symbol lookup call to self
		if err = symbolz.Symbolize("http://"+r.Host+"/symbol", fetch.PostURL, prof); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to symbolize profile: %s\n", err)
		}

		// Create a new profile report
		rpt := report.NewDefault(prof, report.Options{
			OutputFormat:   report.Text,
			CumSort:        cum,
			PrintAddresses: true,
		})

		// Genrate the report, reusing the existing buffer
		buf.Reset()
		report.Generate(&buf, rpt, nil)

		// Send the buffer contents
		w.Write(buf.Bytes())
	}
}

// Trace responds with the execution trace in binary form.
// Tracing lasts for duration specified in seconds GET parameter, or for 1 second if not specified.
// The package initialization registers it as /trace.
func (p *Profiler) Trace(w http.ResponseWriter, r *http.Request) {
	sec, err := strconv.ParseFloat(r.FormValue("seconds"), 64)
	if sec <= 0 || err != nil {
		sec = 1
	}

	// Set Content Type assuming trace.Start will work,
	// because if it does it starts writing.
	w.Header().Set("Content-Type", "application/octet-stream")
	if err := trace.Start(w); err != nil {
		// trace.Start failed, so no writes yet.
		// Can change header back to text content and send error code.
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Could not enable tracing: %s\n", err)
		return
	}
	sleep(w, time.Duration(sec*float64(time.Second)))
	trace.Stop()
}

// Symbol looks up the program counters listed in the request,
// responding with a table mapping program counters to function names.
// The package initialization registers it as /symbol.
func (p *Profiler) Symbol(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// We have to read the whole POST body before
	// writing any output. Buffer the output here.
	var buf bytes.Buffer

	// We don't know how many symbols we have, but we
	// do have symbol information. Pprof only cares whether
	// this number is 0 (no symbols available) or > 0.
	fmt.Fprintf(&buf, "num_symbols: 1\n")

	var b *bufio.Reader

	if r.Method == "POST" {
		b = bufio.NewReader(r.Body)
	} else {
		b = bufio.NewReader(strings.NewReader(r.URL.RawQuery))
	}

	for {
		word, err := b.ReadSlice('+')
		if err == nil {
			word = word[0 : len(word)-1] // trim +
		}
		pc, _ := strconv.ParseUint(string(word), 0, 64)
		if pc != 0 {
			f := runtime.FuncForPC(uintptr(pc))
			if f != nil {
				fmt.Fprintf(&buf, "%#x %s\n", pc, f.Name())
			}
		}

		// Wait until here to check for err; the last
		// symbol will have an err because it doesn't end in +.
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(&buf, "reading request: %v\n", err)
			}
			break
		}
	}

	w.Write(buf.Bytes())
}

func sleep(w http.ResponseWriter, d time.Duration) {
	var clientGone <-chan bool
	if cn, ok := w.(http.CloseNotifier); ok {
		clientGone = cn.CloseNotify()
	}
	select {
	case <-time.After(d):
	case <-clientGone:
	}
}
