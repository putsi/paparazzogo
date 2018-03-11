package main

import (
	"github.com/putsi/paparazzogo"
	"log"
	"net/http"
	"time"
)

func main() {

	// Local server settings
	imgPath1 := "/img1.jpg"
	imgPath2 := "/img2.jpg"
	addr := ":8080"

	// MJPEG-stream settings
	user := ""
	pass := ""
	// If there is zero GET-requests for 30 seconds, mjpeg-stream will be closed.
	// Streaming will be reopened after next request.
	timeout := 30 * time.Second
	mjpegStream1 := "http://webcam.st-malo.com/axis-cgi/mjpg/video.cgi"
	mjpegStream2 := "http://85.157.217.67/axis-cgi/mjpg/video.cgi"

	mjpegHandler1 := paparazzogo.NewMjpegproxy()
	mjpegHandler1.OpenStream(mjpegStream1, user, pass, timeout)

	mjpegHandler2 := paparazzogo.NewMjpegproxy()
	mjpegHandler2.OpenStream(mjpegStream2, user, pass, timeout)

	mux := http.NewServeMux()
	mux.Handle(imgPath1, mjpegHandler1)
	mux.Handle(imgPath2, mjpegHandler2)

	s := &http.Server{
		Addr:    addr,
		Handler: mux,
		// Read- & Write-timeout prevent server from getting overwhelmed in idle connections
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Fatal(s.ListenAndServe())

	block := make(chan bool)
	// time.Sleep(time.Second * 30)
	// mjpegHandler2.CloseStream()
	// mjpegHandler2.OpenStream(newMjpegstream, newUser, newPass, newTimeout)
	<-block

}
