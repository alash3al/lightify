package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"regexp"

	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
	"github.com/tdewolff/minify/html"
	"github.com/tdewolff/minify/js"
	"github.com/tdewolff/minify/json"
	"github.com/tdewolff/minify/svg"
	"github.com/tdewolff/minify/xml"
	"github.com/vulcand/oxy/forward"
)

var (
	flagUpstream = flag.String("upstream", "http://localhost:8080", "the upstream server to fallback")
	flagHTTPAddr = flag.String("http", ":80", "the http port to listen on")
)

func main() {
	flag.Parse()

	m := minify.New()

	m.AddFunc("text/css", css.Minify)
	m.AddFunc("text/html", html.Minify)
	m.AddFunc("text/javascript", js.Minify)
	m.AddFunc("image/svg+xml", svg.Minify)
	m.AddFuncRegexp(regexp.MustCompile("[/+]json$"), json.Minify)
	m.AddFuncRegexp(regexp.MustCompile("[/+]xml$"), xml.Minify)

	forwarder := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var err error
		req.URL, err = url.Parse(*flagUpstream)
		if err != nil {
			http.Error(w, "Internal Server Error", 500)
			return
		}
		fwd, err := forward.New(forward.PassHostHeader(true))
		if err != nil {
			http.Error(w, "Internal Server Error", 500)
			return
		}
		fwd.ServeHTTP(w, req)
	})

	log.Fatal(http.ListenAndServe(*flagHTTPAddr, m.Middleware(forwarder)))
}
