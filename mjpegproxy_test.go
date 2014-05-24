// Copyright 2014 Jarmo Puttonen <jarmo.puttonen@gmail.com>. All rights reserved.
// Use of this source code is governed by a MIT-style
// licence that can be found in the LICENCE file.

/*Package paparazzogo implements a caching proxy for
serving MJPEG-stream as JPG-images.
*/
package paparazzogo

/*
Test coverage: 86.4% of statements.
*/

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

//Multipart body for testing.
var firstPart = "01234567890"
var boundary = "MyBoundary"
var validBody = `
Content-type: multipart/x-mixed-replace;boundary=` + boundary + `

--` + boundary + `
Content-type: text/plain

` + firstPart + `

--` + boundary + `--
   `
var invalidBody = `
Content-type: multipart/x-mixed-replace;boundary=` + boundary + `
--` + boundary + `
Content-type: text/plain
` + firstPart + `
--` + boundary + `--
   `
var malformedPart = `
Content-type: multipart/x-mixed-replace;boundary=` + boundary + `

--` + boundary + `
Content-type: text/plain

` + firstPart + `

--` + boundary + `
   `

var streamBody = `
--` + boundary + `
Content-type: text/plain

` + firstPart + `

--` + boundary + `--
   `

func Test_NewMjpegproxy(t *testing.T) {
	mp := NewMjpegproxy()
	if mp == nil {
		t.Fatal("Could not create Mjpegproxy!")
	}
}

func Test_CloseStream(t *testing.T) {
	mp := NewMjpegproxy()
	mp.CloseStream()
	if mp.running != false {
		t.Fatalf("Wrong run state: %s", mp.running)
	}
}

func Test_setRunning(t *testing.T) {
	mp := NewMjpegproxy()
	mp.setRunning(true)
	if mp.running != true {
		t.Fatalf("Wrong run state: expected true, got %s", mp.running)
	}
	mp.setRunning(false)
	if mp.running != false {
		t.Fatalf("Wrong run state: expected false, got %s", mp.running)
	}
}

func Test_GetRunning(t *testing.T) {
	mp := NewMjpegproxy()
	mp.setRunning(true)
	if mp.GetRunning() != true {
		t.Fatalf("Wrong run state: expected true, got %s", mp.GetRunning())
	}
	mp.setRunning(false)
	if mp.running != false {
		t.Fatalf("Wrong run state: expected false, got %s", mp.GetRunning())
	}
}

func Test_getresponse_valid(t *testing.T) {
	msg := "Test string"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, msg)
	}))
	defer ts.Close()
	mp := NewMjpegproxy()
	request, err := http.NewRequest("GET", ts.URL, nil)
	if err != nil {
		t.Fatal("Failed to create request.")
	}
	res, err := mp.getresponse(request)
	if err != nil {
		t.Fatal(err)
	}
	responsebody, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(responsebody), msg) {
		t.Fatalf("Response body mismatch: %s vs %s", msg, string(responsebody))
	}
}

func Test_getresponse_invalid_status(t *testing.T) {
	invalidstatus := 418
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "", invalidstatus)
	}))
	defer ts.Close()
	mp := NewMjpegproxy()
	request, err := http.NewRequest("GET", ts.URL, nil)
	if err != nil {
		t.Fatal("Failed to create request.")
	}
	_, err = mp.getresponse(request)
	if err == nil {
		t.Fatal("Unexpected nil error!")
	}
	str := strconv.Itoa(invalidstatus)
	if !strings.Contains(err.Error(), str) {
		t.Fatalf("Wrong status code: %s vs %s", err.Error(), str)
	}
}
func Test_getresponse_noconnection(t *testing.T) {
	mp := NewMjpegproxy()
	request, err := http.NewRequest("GET", "http://127.0.0.1:99999/", nil)
	if err != nil {
		t.Fatal("Failed to create request.")
	}
	_, err = mp.getresponse(request)
	if err == nil {
		t.Fatal("Unexpected nil error!")
	}
	if !strings.Contains(err.Error(), "invalid port 99999") {
		t.Fatalf("Wrong error on connection refuse: %s", err.Error())
	}
}

func Test_getmultipart_valid(t *testing.T) {
	mp := NewMjpegproxy()
	response := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Proto:      "HTTP/1.0",
		Header: http.Header{
			"Content-Type": []string{"multipart/x-mixed-replace; boundary=" + boundary},
		},
		Body:          ioutil.NopCloser(bytes.NewBufferString(validBody)),
		ContentLength: int64(len(validBody)),
	}
	_, _, err := mp.getmultipart(response)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_getmultipart_noboundary(t *testing.T) {
	mp := NewMjpegproxy()
	response := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Proto:      "HTTP/1.0",
		Header: http.Header{
			"Content-Type": []string{"multipart/x-mixed-replace"},
		},
		Body:          ioutil.NopCloser(bytes.NewBufferString(validBody)),
		ContentLength: int64(len(validBody)),
	}
	_, _, err := mp.getmultipart(response)
	if err == nil {
		t.Fatal("Unexpected nil error!")
	}
	if !strings.Contains(err.Error(), "No multipart boundary param in Content-Type!") {
		t.Fatalf("Wrong error: %s", err.Error())
	}
}

func Test_getmultipart_invalid(t *testing.T) {
	mp := NewMjpegproxy()
	response := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Proto:      "HTTP/1.0",
		Header: http.Header{
			"Content-Type": []string{"multipart/form-data"},
		},
		Body:          ioutil.NopCloser(bytes.NewBufferString(validBody)),
		ContentLength: int64(len(validBody)),
	}
	_, _, err := mp.getmultipart(response)
	if err == nil {
		t.Fatal("Unexpected nil error!")
	}
	if !strings.Contains(err.Error(), "Wrong Content-Type: expected multipart/x-mixed-replace!, got multipart/form-data") {
		t.Fatalf("Wrong error: %s", err.Error())
	}

}

func Test_getmultipart_noCT(t *testing.T) {
	mp := NewMjpegproxy()
	response := &http.Response{
		Status:        "200 OK",
		StatusCode:    200,
		Proto:         "HTTP/1.0",
		Header:        http.Header{},
		Body:          ioutil.NopCloser(bytes.NewBufferString(validBody)),
		ContentLength: int64(len(validBody)),
	}
	_, _, err := mp.getmultipart(response)
	if err == nil {
		t.Fatal("Unexpected nil error!")
	}
	if !strings.Contains(err.Error(), "Content-Type isn't specified!") {
		t.Fatalf("Wrong error: %s", err.Error())
	}

}

func Test_readpart_valid(t *testing.T) {
	mp := NewMjpegproxy()
	mp.setRunning(true)
	reader := io.ReadCloser(ioutil.NopCloser(bytes.NewBufferString(validBody)))
	mpart := multipart.NewReader(reader, boundary)

	bytes, err := mp.readpart(mpart)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(bytes.String(), firstPart) {
		t.Fatalf("Bad part: %s vs %s", bytes.String(), firstPart)
	}
}

func Test_readpart_invalid(t *testing.T) {
	mp := NewMjpegproxy()
	mp.setRunning(true)
	reader := io.ReadCloser(ioutil.NopCloser(bytes.NewBufferString(invalidBody)))
	mpart := multipart.NewReader(reader, boundary)

	_, err := mp.readpart(mpart)
	if err == nil {
		t.Fatal("Unexpected nil error!")
	}
	if !strings.Contains(err.Error(), "malformed MIME header line:") {
		t.Fatal(err)
	}
}

func Test_readpart_parterror(t *testing.T) {
	mp := NewMjpegproxy()
	mp.setRunning(true)
	reader := io.ReadCloser(ioutil.NopCloser(bytes.NewBufferString(malformedPart)))
	mpart := multipart.NewReader(reader, boundary)

	_, err := mp.readpart(mpart)
	if err != nil {
		t.Fatal(err)
	}
	_, err = mp.readpart(mpart)
	if err == nil {
		t.Fatal("Unexpected nil error!")
	}
	if err.Error() != "unexpected EOF" {
		t.Fatal(err)
	}
}

func Test_ServeHTTP(t *testing.T) {
	mp := NewMjpegproxy()
	req := &http.Request{}
	testString := []byte("Test String")
	mp.curImg.Write(testString)
	recorder := httptest.NewRecorder()
	mp.ServeHTTP(recorder, req)
	if time.Since(mp.lastConn) > time.Second {
		t.Fatal("Unexpected lastconn value!")
	}
	if !bytes.Equal(recorder.Body.Bytes(), testString) {
		t.Fatalf("Content mismatch: expected %s, got %s", string(testString), string(mp.curImg.Bytes()))
	}
}

func Test_OpenStream_logic(t *testing.T) {
	user := "user"
	pass := "pass"
	mp := NewMjpegproxy()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "multipart/x-mixed-replace;boundary="+boundary)
		fmt.Fprintln(w, streamBody)
	}))
	defer ts.Close()
	mp.setRunning(true)
	mp.OpenStream(ts.URL, user, pass, time.Second)
	defer mp.CloseStream()
	mp.conChan <- time.Now()
	time.Sleep(time.Millisecond)
	if !strings.Contains(mp.curImg.String(), firstPart) {
		t.Fatalf("Wrong response: expected %s, got %s", firstPart, mp.curImg.String())
	}
	mp.setRunning(false)
}
