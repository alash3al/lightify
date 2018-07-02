Lightify
=========
> Let's minify, compress &amp; combine static files on the fly.

![](http://preview.ibb.co/nkCgpy/36283128_1069026009937885_5431723196539731968_o_1.jpg)

Features
==========
- Simple &amp; Stupid.
- Minify `css, js, html, xml, json`.
- Gzip each minified file.
- Combine and inline self-hosted `css, js` on the fly!
- Automatically fixes internal relative css imports!
- Supports any http upstream.
- Portable, no dependancies.

Download
========
- Docker `$ docker pull alash3al/lightify`
- Binaries go to [releases page](https://github.com/alash3al/lightify/releases)
- From Source `go get github.com/alash3al/lightify`

Usage
======
- Docker `$ docker run --network=host alash3al/lightify --upstream="http://localhost:8080"`
- Binaries `./lightify -http :80 -upstream http://localhost:8080`
- From Source `lightify -http :80 -upstream http://localhost:8080`

Help
====
- Docker `$ docker run alash3al/lightify --help`
- Binaries `./lightify --help`
- From Source `lightify --help`

Credits
========
Mohamed Al Ashaal, a Gopher ;)

License
========
MIT License

Contribution
=============
The door is always open ;)