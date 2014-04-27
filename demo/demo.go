package main

import (
	"github.com/putsi/paparazzogo"
	"log"
	"net/http"
	"time"
)

func main() {

	// Local server settings
	imgPath := "/img.jpg"
	addr := ":8080"

	// MJPEG-stream settings
	user := ""
	pass := ""
	// If there is zero GET-requests for 5 seconds, mjpeg-stream will be closed.
	// Streaming will be reopened after next request.
	timeout := 5 * time.Second
	mjpegStream := "http://westunioncam.studentaffairs.duke.edu/mjpg/video.mjpg"

	mjpegHandler := paparazzogo.NewMjpegproxy()
	mjpegHandler.OpenStream(mjpegStream, user, pass, timeout)

	http.Handle(imgPath, mjpegHandler)
	log.Fatal(http.ListenAndServe(addr, nil))

	block := make(chan bool)
	// time.Sleep(time.Second * 30)
	// mp.CloseStream()
	// mp.OpenStream(newMjpegstream, newUser, newPass, newTimeout)
	<-block

}
