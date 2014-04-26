package main

import (
	"github.com/putsi/paparazzo.go"
	"time"
)

func main() {

	port := ":8080"
	user := ""
	pass := ""
	timeout := 5 * time.Second
	mjpegStream := "http://westunioncam.studentaffairs.duke.edu/mjpg/video.mjpg"
	imgPath := "/img.jpg"

	mp := mjpegproxy.NewMjpegproxy()

	mp.StartCrawling(mjpegStream, user, pass, timeout)
	mp.Serve(imgPath, port)

	block := make(chan bool)

	// time.Sleep(time.Second * 30)
	// mp.StopServing()
	// mp.StopCrawling()
	<-block

}
