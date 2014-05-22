// Copyright 2014 Jarmo Puttonen <jarmo.puttonen@gmail.com>. All rights reserved.
// Use of this source code is governed by a MIT-style
// licence that can be found in the LICENCE file.

/*Package paparazzogo implements a caching proxy for
serving MJPEG-stream as JPG-images.
*/
package paparazzogo

import (
	"bytes"
	"errors"
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
	buf := bytes.Buffer{}

	m.curImgLock.RLock()
	buf.Write(m.curImg.Bytes())
	m.curImgLock.RUnlock()

	w.Write(buf.Bytes())
	select {
	case m.conChan <- time.Now():
	default:
		m.lastConnLock.Lock()
		m.lastConn = time.Now()
		m.lastConnLock.Unlock()
	}
}

// CloseStream stops and closes MJPEG-stream.
func (m *Mjpegproxy) CloseStream() {
	m.setRunning(false)
}

// OpenStream creates a go-routine of openstream.
func (m *Mjpegproxy) OpenStream(mjpegStream, user, pass string, timeout time.Duration) {
	go m.openstream(mjpegStream, user, pass, timeout)
}

// GetRunning returns state of openstream.
func (m *Mjpegproxy) GetRunning() bool {
	m.runningLock.RLock()
	defer m.runningLock.RUnlock()
	return m.running
}

func (m *Mjpegproxy) setRunning(r bool) {
	m.runningLock.Lock()
	defer m.runningLock.Unlock()
	m.running = r
}

func (m *Mjpegproxy) getresponse(request *http.Request) (*http.Response, error) {
	tr := &http.Transport{DisableKeepAlives: true}
	client := &http.Client{Transport: tr}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != 200 {
		response.Body.Close()
		errs := "Got invalid response status: " + response.Status
		return nil, errors.New(errs)
	}
	return response, nil
}

func (m *Mjpegproxy) getmultipart(response *http.Response) (io.ReadCloser, *string, error) {
	// Get boundary string from response and clean it up
	boundary := response.Header.Get("Content-Type")
	if boundary == "" {
		return nil, nil, errors.New("Found no boundary-value in response!")
	}
	split := strings.Split(boundary, "boundary=")
	boundary = split[1]
	// TODO: Find out what happens when boundarystring is "--something--" or "something--"
	boundary = strings.Replace(boundary, "--", "", 1)
	reader := io.ReadCloser(response.Body)
	return reader, &boundary, nil
}

func (m *Mjpegproxy) readpart(mpread *multipart.Reader) (*bytes.Buffer, error) {
	buffer := make([]byte, m.partbufsize)
	img := bytes.Buffer{}
	part, err := mpread.NextPart()
	if err != nil {
		return nil, err
	}
	defer part.Close()
	amnt := 0
	// Get frame parts until err is EOF or running is false
	for err == nil && m.GetRunning() {
		amnt, err = part.Read(buffer)
		if err != nil && err.Error() != "EOF" {
			return nil, err
		}
		img.Write(buffer[0:amnt])
	}
	return &img, nil
}

// OpenStream sends request to target and handles
// response. It opens MJPEG-stream and copies received
// frame to m.curImg. It closes stream if m.CloseStream()
// is called or if difference between current time and
// time of last request to ServeHTTP is bigger than timeout.
func (m *Mjpegproxy) openstream(mjpegStream, user, pass string, timeout time.Duration) {
	var lastconn time.Time
	var img *bytes.Buffer
	// How long will we wait after error before trying to reconnect
	waittime := time.Second * 5
	m.setRunning(true)
	request, err := http.NewRequest("GET", mjpegStream, nil)
	if user != "" && pass != "" {
		request.SetBasicAuth(user, pass)
	}
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Starting streaming from", mjpegStream)

	for m.GetRunning() {
		lastconn = <-m.conChan
		if !m.GetRunning() || (time.Since(lastconn) > timeout) {
			continue
		}
		var response *http.Response
		response, err = m.getresponse(request)
		if err != nil {
			log.Println(err)
			time.Sleep(waittime)
			continue
		}
		defer response.Body.Close()
		reader, boundary, err := m.getmultipart(response)
		if err != nil {
			log.Println(err)
			response.Body.Close()
			time.Sleep(waittime)
			continue
		}
		defer reader.Close()
		mpread := multipart.NewReader(reader, *boundary)

		for m.GetRunning() && (time.Since(lastconn) < timeout) && err == nil {
			m.lastConnLock.RLock()
			lastconn = m.lastConn
			m.lastConnLock.RUnlock()
			img, err = m.readpart(mpread)
			if err != nil {
				log.Println(err)
				reader.Close()
				response.Body.Close()
				time.Sleep(waittime)
				break
			}
			m.curImgLock.Lock()
			m.curImg.Reset()
			_, err = m.curImg.Write(img.Bytes())
			m.curImgLock.Unlock()
			img.Reset()
			if err != nil {
				log.Println(err)
				reader.Close()
				response.Body.Close()
				time.Sleep(waittime)
				break
			}
		}
	}
	log.Println("Stopped streaming from", mjpegStream)
}
