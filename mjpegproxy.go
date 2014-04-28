// Copyright 2014 Jarmo Puttonen <jarmo.puttonen@gmail.com>. All rights reserved.
// Use of this source code is governed by a MIT-style
// licence that can be found in the LICENCE file.

/* Package paparazzogo implements a caching proxy for
serving MJPEG-stream as JPG-images.
*/
package paparazzogo

import (
	"bytes"
	"io"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// A Mjpegproxy implements http.Handler	interface and generates
// JPG-images from a MJPEG-stream.
type Mjpegproxy struct {
	partbufsize int
	imgbufsize  int

	curImg       bytes.Buffer
	curImgLock   sync.RWMutex
	conChan      chan time.Time
	lastConn     time.Time
	lastConnLock sync.RWMutex
	running      bool
	runningLock  sync.RWMutex
	l            net.Listener
	writer       io.Writer
	handler      http.Handler
}

// NewMjpegproxy returns a new Mjpegproxy with default buffer
// sizes.
func NewMjpegproxy() *Mjpegproxy {
	p := &Mjpegproxy{
		conChan: make(chan time.Time),
		// Max MJPEG-frame part size 1Mb.
		partbufsize: 125000,
		// Max MJPEG-frame size 5Mb.
		imgbufsize: 625000,
	}
	return p
}

// ServeHTTP uses w to serve current last MJPEG-frame
// as JPG. It also reopens MJPEG-stream
// if it was closed by idle timeout.
func (m *Mjpegproxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	m.curImgLock.RLock()
	w.Write(m.curImg.Bytes())
	m.curImgLock.RUnlock()

	m.lastConnLock.Lock()
	m.lastConn = time.Now()
	m.lastConnLock.Unlock()

	select {
	case m.conChan <- time.Now():
	default:
	}
}

// CloseStream stops and closes MJPEG-stream.
func (m *Mjpegproxy) CloseStream() {
	m.runningLock.Lock()
	m.running = false
	m.runningLock.Unlock()
}

// OpenStream creates a go-routine of openstream.
func (m *Mjpegproxy) OpenStream(mjpegStream, user, pass string, timeout time.Duration) {
	go m.openstream(mjpegStream, user, pass, timeout)
}

// checkrunning returns current running state.
func (m *Mjpegproxy) checkrunning() bool {
	m.runningLock.RLock()
	running := m.running
	m.runningLock.RUnlock()
	if running {
		return true
	} else {
		return false
	}
}

// OpenStream sends request to target and handles
// response. It opens MJPEG-stream and received frame
// to m.curImg. It closes stream if m.running is set
// to false or if difference between current time and
// lastconn (time of last request to ServeHTTP)
// is bigger than timeout.
func (m *Mjpegproxy) openstream(mjpegStream, user, pass string, timeout time.Duration) {
	m.runningLock.Lock()
	m.running = true
	m.runningLock.Unlock()
	request, err := http.NewRequest("GET", mjpegStream, nil)
	if user != "" && pass != "" {
		request.SetBasicAuth(user, pass)
	}
	if err != nil {
		log.Fatal(err)
	}

	buffer := make([]byte, m.partbufsize)
	img := bytes.Buffer{}

	var lastconn time.Time
	tr := &http.Transport{DisableKeepAlives: true}
	client := &http.Client{Transport: tr}

	for m.checkrunning() {
		//TODO2: change this to something that uses m.lastConn
		lastconn = <-m.conChan
		if !m.checkrunning() || (time.Since(lastconn) > timeout) {
			continue
		}
		func() {
			var response *http.Response
			response, err = client.Do(request)
			log.Println("New response")
			if err != nil {
				log.Println(err.Error())
				return
			}
			defer response.Body.Close()
			if response.StatusCode == 503 {
				log.Println(response.Status)
				return
			}
			if response.StatusCode != 200 {
				log.Fatalln("Got invalid response status: ", response.Status)
			}
			// Get boundary string from response and clean it up
			split := strings.Split(response.Header.Get("Content-Type"), "boundary=")
			boundary := split[1]
			// TODO: Find out what happens when boundarystring ends in --
			boundary = strings.Replace(boundary, "--", "", 1)

			reader := io.ReadCloser(response.Body)
			defer reader.Close()
			mpread := multipart.NewReader(reader, boundary)
			var part *multipart.Part

			for m.checkrunning() && (time.Since(lastconn) < timeout) {
				m.lastConnLock.RLock()
				lastconn = m.lastConn
				m.lastConnLock.RUnlock()
				func() {
					part, err = mpread.NextPart()
					log.Println("New part")
					defer part.Close()
					if err != nil {
						log.Fatal(err)
					}
					// Get frame parts until err is EOF or running is false
					if img.Len() > 0 {
						img.Reset()
					}
					for err == nil && m.checkrunning() {
						amnt := 0
						amnt, err = part.Read(buffer)
						if err != nil && err.Error() != "EOF" {
							log.Fatal(err)
						}
						img.Write(buffer[0:amnt])
					}
					err = nil
					if img.Len() > m.imgbufsize {
						img.Truncate(m.imgbufsize)
					}
					m.curImgLock.Lock()
					defer m.curImgLock.Unlock()
					m.curImg.Reset()
					_, err = m.curImg.Write(img.Bytes())
					if err != nil {
						log.Fatal(err)
					}
				}()
			}
		}()
	}
}
