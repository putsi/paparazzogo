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
	timeout := 5 * time.Second
	mjpegStream1 := "http://westunioncam.studentaffairs.duke.edu/mjpg/video.mjpg"
	mjpegStream2 := "http://axis-m10-webcam.is.kent.edu/axis-cgi/mjpg/video.cgi"

	mjpegHandler1 := paparazzogo.NewMjpegproxy()
	mjpegHandler1.StartCrawling(mjpegStream1, user, pass, timeout)

	mjpegHandler2 := paparazzogo.NewMjpegproxy()
	mjpegHandler2.StartCrawling(mjpegStream2, user, pass, timeout)

	mux := http.NewServeMux()
	mux.Handle(imgPath1, mjpegHandler1)
	mux.Handle(imgPath2, mjpegHandler2)

	log.Fatal(http.ListenAndServe(addr, mux))

	block := make(chan bool)
	// time.Sleep(time.Second * 30)
	// mjpegHandler2.StopCrawling()
	// mjpegHandler2.startCrawling(newMjpegstream, newUser, newPass, newTimeout)
	<-block

}
