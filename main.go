package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
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
	flagMinify   = flag.String("minify", "js,css,json,xml,html,svg", "the types to be minified, empty means none")
	flagCombine  = flag.Bool("combine", true, "combine the assets files on the fly")
	flagGZIP     = flag.Bool("gzip", true, "compress the output")
	flagLog      = flag.Bool("log", true, "enable logging")
)

func main() {
	flag.Parse()

	minifiable := strings.Split(*flagMinify, ",")
	m := minify.New()

	if inArray(minifiable, "css") {
		m.AddFunc("text/css", css.Minify)
	}

	if inArray(minifiable, "html") {
		m.Add("text/html", &html.Minifier{
			KeepConditionalComments: true,
			KeepEndTags:             true,
			KeepDocumentTags:        true,
			KeepDefaultAttrVals:     true,
		})
	}

	if inArray(minifiable, "js") {
		m.AddFunc("text/javascript", js.Minify)
		m.AddFunc("application/javascript", js.Minify)
	}

	if inArray(minifiable, "svg") {
		m.AddFunc("image/svg+xml", svg.Minify)
	}

	if inArray(minifiable, "xml") {
		m.AddFuncRegexp(regexp.MustCompile("[/+]xml$"), xml.Minify)
	}

	if inArray(minifiable, "json") {
		m.AddFuncRegexp(regexp.MustCompile("[/+]json$"), json.Minify)
	}

	cssURLs := regexp.MustCompile(`(url|\@import)\((.*?)\)`)
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
			w.Header.Del("Content-Length")

			if strings.ToLower(w.Header.Get("Content-Encoding")) == "gzip" {
				w.Body, err = gzip.NewReader(w.Body)
				if err != nil {
					return nil
				}
				w.Header.Del("Content-Encoding")
			}

			chunk := make([]byte, 512)
			w.Body.Read(chunk)

			w.Body = ioutil.NopCloser(io.MultiReader(bytes.NewBuffer(chunk), w.Body))

			if !strings.Contains(strings.ToLower(http.DetectContentType(chunk)), "text/html") {
				return nil
			}

			doc, err := goquery.NewDocumentFromReader(w.Body)
			if err != nil {
				return nil
			}

			if *flagCombine {
				// bundle css
				doc.Find("link").Each(func(_ int, s *goquery.Selection) {
					dst := s.AttrOr("href", "")
					if dst == "" {
						return
					}
					if s.AttrOr("rel", "") != "stylesheet" {
						return
					}
					dst = fixURL(dst, w.Request.Host)
					u, err := url.Parse(dst)
					if err != nil {
						return
					}
					if d := fetch(dst); d != "" {
						for _, val := range cssURLs.FindAllStringSubmatch(d, -1) {
							newURL := strings.Trim(val[2], `"'`)
							if !strings.HasPrefix(newURL, "//") && !strings.HasPrefix(newURL, "http://") && !strings.HasPrefix(newURL, "http://") && !strings.HasPrefix(newURL, "/") && !strings.HasPrefix(newURL, "data:") {
								newURL = "//" + u.Host + path.Join("/", path.Dir(u.Path), newURL) + "?" + u.RawQuery
							}
							if val[2] == newURL {
								continue
							}
							if val[1] == "url" {
								d = strings.Replace(d, "url("+val[2]+")", "url("+newURL+")", -1)
							} else if val[1] == "@import" {
								d = strings.Replace(d, "@import("+val[2]+")", "@import("+newURL+")", -1)
							}
						}
						s.ReplaceWithHtml("<style>" + (d) + "</style>")
					}
				})

				// bundleJS
				doc.Find("script").Each(func(_ int, s *goquery.Selection) {
					dst := s.AttrOr("src", "")
					if dst == "" {
						return
					}
					// s.SetAttr("async", "true")
					dst = fixURL(dst, w.Request.Host)
					if d := fetch(dst); d != "" {
						s.RemoveAttr("src")
						s.SetText(d)
					}
				})
			}

			html, _ := doc.Html()
			w.Body = ioutil.NopCloser(strings.NewReader(html))

			return nil
		}))

		if err != nil {
			http.Error(w, "Internal Server Error", 500)
			return
		}

		fwd.ServeHTTP(w, req)
	})

	var container http.Handler = forwarder

	if *flagMinify != "" {
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
	if !strings.HasPrefix(dst, "//") && !strings.HasPrefix(dst, "http://") && !strings.HasPrefix(dst, "https://") {
		dst = "//" + host + "/" + strings.TrimLeft(dst, "/")
	}
	return dst
}

func inArray(a []string, s string) bool {
	for _, v := range a {
		if s == v {
			return true
		}
	}
	return false
}
