package main

import (
	"compress/gzip"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/handlers"
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
	flagMinify   = flag.Bool("minify", true, "minify the assets files on the fly")
	flagCombine  = flag.Bool("combine", true, "combine the assets files on the fly")
	flagGZIP     = flag.Bool("gzip", true, "compress the output")
	flagLog      = flag.Bool("log", true, "enable logging")
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

		fwd, err := forward.New(forward.PassHostHeader(true), forward.ResponseModifier(func(w *http.Response) (err error) {
			w.Header.Del("Server")
			w.Header.Del("X-Powered-By")

			if strings.ToLower(w.Header.Get("Content-Encoding")) == "gzip" {
				w.Body, err = gzip.NewReader(w.Body)
				if err != nil {
					return nil
				}
				w.Header.Del("Content-Encoding")
			}

			if !*flagCombine || !strings.Contains(strings.ToLower(w.Header.Get("Content-Type")), "text/html") {
				return nil
			}

			doc, err := goquery.NewDocumentFromReader(w.Body)
			if err != nil {
				return err
			}

			bundleCSS := ""
			doc.Find("link").Each(func(_ int, s *goquery.Selection) {
				dst := s.AttrOr("href", "")
				if dst == "" {
					return
				}
				dst = fixURL(dst, w.Request.Host)
				u, err := url.Parse(dst)
				if err != nil || !strings.HasSuffix(u.Path, "css") {
					return
				}
				if u.Host != w.Request.Host {
					return
				}
				if d := fetch(dst); d != "" {
					bundleCSS += d
					s.Remove()
				}
			})

			bundleJS := ""
			doc.Find("script").Each(func(_ int, s *goquery.Selection) {
				dst := s.AttrOr("src", "")
				if dst == "" {
					return
				}
				dst = fixURL(dst, w.Request.Host)
				u, err := url.Parse(dst)
				if err != nil {
					return
				}
				if u.Host != w.Request.Host {
					return
				}
				if js := fetch(dst); js != "" {
					bundleJS += js
					s.Remove()
				}
			})

			if bundleCSS != "" {
				doc.Find("head").AppendHtml("<style>" + (bundleCSS) + "</style>")
			}

			if bundleJS != "" {
				doc.Find("body").AppendHtml("<script>" + (bundleJS) + "</script>")
			}

			html, _ := doc.Html()
			w.Body = ioutil.NopCloser(strings.NewReader(html))
			w.Header.Set("Content-Length", strconv.Itoa(len([]rune(html))))

			return nil
		}))

		if err != nil {
			http.Error(w, "Internal Server Error", 500)
			return
		}

		fwd.ServeHTTP(w, req)
	})

	var container http.Handler = forwarder

	if *flagMinify {
		container = m.Middleware(container)
	}

	if *flagGZIP {
		container = handlers.CompressHandlerLevel(container, 9)
	}

	if *flagLog {
		container = handlers.CombinedLoggingHandler(os.Stdout, container)
	}

	log.Println("> starting server on `" + (*flagHTTPAddr) + "`")
	log.Fatal(http.ListenAndServe(*flagHTTPAddr, container))
}

func fetch(dst string) string {
	resp, err := http.Get(dst)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	d, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return string(d)
}

func fixURL(dst, host string) string {
	if strings.HasPrefix(dst, "//") {
		dst = "http:" + dst
	}
	if !strings.HasPrefix(dst, "http://") && !strings.HasPrefix(dst, "https://") {
		dst = "http://" + host + "/" + strings.TrimLeft(dst, "/")
	}
	return dst
}
