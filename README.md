# Extended profiling / debugging web interface

Introduction
------------

The profiler package implements profiling features and a web frontend for easy access during runtime. It is based on the Go-inherent [net/http/pprof](https://golang.org/pkg/net/http/pprof/) package, but extends it by some additional features. Unlike [net/http/pprof](https://golang.org/pkg/net/http/pprof/), the profiler package usage is not implicit by mere inclusion for the sake of its side effects, but explicit, i.e. an instance of a *Profiler* type has to be created, thus allowing better control over its parameters. Further differences w.r.t. [net/http/pprof](https://golang.org/pkg/net/http/pprof/) are:
* The web frontend is always run using a dedicated HTTP server instance to avoid interference with any web server potentially served by the calling package (including accidental exposure of the debugging interface on a public web server)
* Several functional options are available, allowing to set various parameters in an easy and efficient manner (e.g. a custom HTTP server or even any custom / additional middleware)
* Direct access to plain text CPU profiles during runtime (i.e. without the need to download a binary profile and the use of e.g. *go tool pprof ...*)
* Slightly improved web interface

Installation and usage
----------------------

The import path for the package is *github.com/fako1024/profiler*.

To install it, run:

    go get github.com/fako1024/profiler

The most trivial use case is to import the package and start up the profiler with default options in the background:

```Go
...
import "github.com/fako1024/profiler"

func main() {

  // Makes profiler available on http://127.0.0.1:6060/
  profiler.New().Run()

  ...
}
```

More granular control is possible via functional option calls (all available options can be found in [options.go](options.go)):

```Go
...
import "github.com/fako1024/profiler"

// Custom template, only displaying the basic profiles
const customTemplate = `<html>
<head>
<title>Go Custom Profiling interface</title>
</head>
<body>
<b>Go Custom Profiling interface</b><br>
<br>
<table>
{{range .}}
<tr><td align=left><a href="{{.Name}}?debug=1">{{.Name}}</a><td> ({{.Count}})
{{end}}
</table>
</body>
</html>
`

// Custom middleware, logging all request URLs to STDOUT
func customMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Print the request URL
		fmt.Println(r.URL.String())

		// Serve the next layer(s) of middleware
		next.ServeHTTP(w, r)
	})
}

func main() {

	profiler.New(
    profiler.WithAddr("127.0.0.1:6061"),                  // Start web interface on port 6061
		profiler.WithKeyPair("/tmp/tls.crt", "/tmp/tls.key"), // Enable HTTPS (certificate / key)
		profiler.WithMiddleware(customMiddleware),            // Enable request logging to STDOUT
		profiler.WithHTMLTemplate(customTemplate),            // Set custom HTML index page template
		profiler.WithErrorHandler(func(err error) {           // Set error handler (panic in case http.ListenAndServe() fails)
			panic(err)
		}),
	).Run()

  ...
}
```

License
-------

The profiler package is governed under a BSD-style license. Please see the LICENSE file for details.
