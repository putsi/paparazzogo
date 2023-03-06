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
	"mime"
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
	partbufsize      int64
	waittime         time.Duration
	responseduration time.Duration
	caching          bool

	mjpegStream  string
	curImg       bytes.Buffer
	curImgLock   sync.RWMutex
	conChan      chan time.Time
	lastConn     time.Time
	lastConnLock sync.RWMutex
	lastModified time.Time
	lastModLock  sync.RWMutex
	running      bool
	runningLock  sync.RWMutex
	l            net.Listener
	writer       io.Writer
	handler      http.Handler
}

// NewMjpegproxy returns a new Mjpegproxy with default values.
func NewMjpegproxy() *Mjpegproxy {
	p := &Mjpegproxy{
		// Max MJPEG-frame size 5Mb.
		partbufsize: 625000,
		// Sleep time between error and reconnecting to stream.
		waittime: time.Second * 1,
		// How long to use one stream response before reconnecting.
		responseduration: time.Hour,
		// Caching enables/disables support for client-side caching
		// of jpg-files. If enabled, saves bandwidth.
		// If disabled, allows more than one frame per second.
		caching: false,
	}
	return p
}

// ServeHTTP uses w to serve current last MJPEG-frame
// as JPG. It also reopens MJPEG-stream
// if it was closed by idle timeout.
func (m *Mjpegproxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	select {
	case m.conChan <- time.Now():
	default:
		m.lastConnLock.Lock()
		m.lastConn = time.Now()
		m.lastConnLock.Unlock()
	}
	buf := bytes.Buffer{}
	m.curImgLock.RLock()
	buf.Write(m.curImg.Bytes())
	m.curImgLock.RUnlock()

	reader := bytes.NewReader(buf.Bytes())
	if reader == nil {
		log.Println(m.mjpegStream, "ServeHTTP could not create bytes.Reader!")
		return
	}
	if !m.caching {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		w.Header().Set("Content-Length", reader.Len())
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		reader.WriteTo(w)
	} else {
		m.lastModLock.RLock()
		modtime := m.lastModified
		m.lastModLock.RUnlock()
		if modtime.String() == "" {
			modtime = time.Now()
		}
		http.ServeContent(w, req, "img.jpg", modtime, reader)
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

func (m *Mjpegproxy) getboundary(response *http.Response) (string, error) {
	header := response.Header.Get("Content-Type")
	if header == "" {
		return "", errors.New("Content-Type isn't specified!")
	}
	ct, params, err := mime.ParseMediaType(header)
	if err != nil {
		return "", err
	}
	if ct != "multipart/x-mixed-replace" {
		errs := "Wrong Content-Type: expected multipart/x-mixed-replace, got " + ct
		return "", errors.New(errs)
	}
	boundary, ok := params["boundary"]
	if !ok {
		return "", errors.New("No multipart boundary param in Content-Type!")
	}
	// Some IP-cameras screw up boundary strings so we
	// have to remove excessive "--" characters manually.
	boundary = strings.Replace(boundary, "--", "", -1)
	return boundary, nil
}

// OpenStream sends request to target and handles
// response. It opens MJPEG-stream and copies received
// frame to m.curImg. It closes stream if m.CloseStream()
// is called or if difference between current time and
// time of last request to ServeHTTP is bigger than timeout.
func (m *Mjpegproxy) openstream(mjpegStream, user, pass string, timeout time.Duration) {
	m.setRunning(true)
	m.conChan = make(chan time.Time)
	m.mjpegStream = mjpegStream
	var lastconn time.Time
	var img *multipart.Part

	request, err := http.NewRequest("GET", mjpegStream, nil)
	if err != nil {
		log.Fatal(m.mjpegStream, err)
	}
	if user != "" && pass != "" {
		request.SetBasicAuth(user, pass)
	}
	var response *http.Response
	var boundary string
	var mpread *multipart.Reader
	var starttime time.Time
	buf := new(bytes.Buffer)

	log.Println("Starting streaming from", mjpegStream)

	for m.GetRunning() {
		lastconn = <-m.conChan
		m.lastConnLock.Lock()
		m.lastConn = lastconn
		m.lastConnLock.Unlock()
		if !m.GetRunning() {
			continue
		}

		response, err = m.getresponse(request)
		if err != nil {
			log.Println(m.mjpegStream, err)
			time.Sleep(m.waittime)
			continue
		}
		starttime = time.Now()
		boundary, err = m.getboundary(response)

		if err != nil {
			log.Println(m.mjpegStream, err)
			response.Body.Close()
			time.Sleep(m.waittime)
			continue
		}
		mpread = multipart.NewReader(response.Body, boundary)
		for m.GetRunning() && (time.Since(lastconn) < timeout) && err == nil {
			if time.Since(starttime) > m.responseduration {
				break
			}
			if time.Since(lastconn) > timeout/2 {
				m.lastConnLock.RLock()
				lastconn = m.lastConn
				m.lastConnLock.RUnlock()
			}
			img, err = mpread.NextPart()
			if err != nil {
				log.Println(m.mjpegStream, err)
				break
			}
			// buf is an additional buffer that allows
			// serving curImg while loading next part.
			buf.Reset()
			_, err = buf.ReadFrom(io.LimitReader(img, m.partbufsize))
			if err != nil {
				img.Close()
				log.Println(m.mjpegStream, err)
				break
			}
			if m.caching {
				m.lastModLock.Lock()
				m.lastModified = time.Now().UTC()
				m.lastModLock.Unlock()
			}
			m.curImgLock.Lock()
			m.curImg.Reset()
			_, err = m.curImg.ReadFrom(buf)
			m.curImgLock.Unlock()
			img.Close()
			if err != nil {
				log.Println(m.mjpegStream, err)
				break
			}
		}
		response.Body.Close()
		time.Sleep(m.waittime)
	}
	log.Println("Stopped streaming from", mjpegStream)
}
