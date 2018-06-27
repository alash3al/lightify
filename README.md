Lightify
=========
> Let's Minify &amp; Compress static files on the fly.

Features
==========
- Simple &amp; Stupid
- Minify `css, js, html, xml, json`
- Gzip each minified file
- Combine and inline self-hosted `css, js` on the fly!
- Supports any http upstream.

Usage
======
```bash
# move your current webserver "nginx/apache ... etc" from port "80" to i.e "8080"
# let lightify listen on port 80 and forward the trafic to the 8080 like this:
$ docker run --network=host alash3al/lightify --upstream="http://localhost:8080"
```