package main

import (
	"github.com/putsi/paparazzogo"
	"time"
)

func main() {

	// Local server settings
	imgPath := "/img.jpg"
	addr := ":8080"

	// MJPEG-stream settings
	user := ""
	pass := ""
	timeout := 5 * time.Second
	mjpegStream := "http://westunioncam.studentaffairs.duke.edu/mjpg/video.mjpg"

	mp := paparazzogo.NewMjpegproxy()

	mp.StartCrawling(mjpegStream, user, pass, timeout)
	mp.Serve(imgPath, addr)

	block := make(chan bool)

	// time.Sleep(time.Second * 30)
	// mp.StopServing()
	// mp.StopCrawling()
	<-block

}
