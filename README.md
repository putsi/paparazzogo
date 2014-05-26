Paparazzo.go
-

_A stalker of [IP cameras](http://en.wikipedia.org/wiki/IP_camera)_

[![endorse](http://api.coderwall.com/putsi/endorsecount.png)](http://coderwall.com/putsi)

**What is this?**

A high performance caching web proxy for serving [MJPG](http://en.wikipedia.org/wiki/Motion_JPEG) streams to the masses.

***Features***

  - Easy to use.
  - Done with [Go programming language](http://golang.org/).
  - Compatible with [http.HandlerFunc](http://golang.org/pkg/net/http/#HandlerFunc).
  - No unnecessary network traffic to IP-camera.

IPCamera (1) <-> (1) Paparazzo.go (1) <-> (N) Users

![Demo screenshot](https://raw.githubusercontent.com/putsi/paparazzogo/master/mjpg_demo.gif "Streaming a VIVOTEK camera")

Background
-

**IP cameras can't handle web traffic**

IP cameras are slow devices that can't handle a regular amount of web traffic. So if you plan to go public with an IP camera you have the following options:

1. **The naive approach** - Embed the camera service directly in your site, e.g. http://201.166.63.44/axis-cgi/jpg/image.cgi?resolution=CIF.
2. **Ye olde approach** - Serve images as static files in your server. I've found that several sites use this approach through messy PHP background jobs that update this files at slow intervals, generating excessive (and unnecessary) disk accesses.
3. **Plug n' pray approach** - Embed a flash or Java-based player, such as the  [Cambozola](http://www.charliemouse.com/code/cambozola/) player. This requires plugins.
4. **MJPG proxy** - Serve the MJPG stream directly if you are targeting only grade A browsers, (sorry IE).
5. **Paparazzo.go: A web service of dynamic images** - Build a MJPG proxy server which parses the stream, updates images in memory, and delivers new images on demand. This approach is scalable, elegant, blazing fast and doesn't require disk access.

Usage
-

Get Paparazzo and start demo:
```
go get github.com/putsi/paparazzogo

cd $GOPATH/src/github.com/putsi/paparazzogo/demo
go run demo.go
open demo.html
```

**See more examples in demo-folder.**

Licence
- 
Use of this source code is governed by a MIT-style licence that can be found in the [LICENCE](https://raw.githubusercontent.com/putsi/paparazzogo/master/LICENSE)-file.

See Also
-
**[The original Paparazzo.js for NodeJS!](https://github.com/rodowi/Paparazzo.js)**