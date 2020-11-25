package main

import (
	"bufio"
	"log"
	"net"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	Status int
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	r.Status = 200
	return r.ResponseWriter.(http.Hijacker).Hijack()
}
func (r *statusRecorder) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}

type Logger struct {
	handler http.Handler
}

func (l *Logger) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	start := time.Now()
	recorder := &statusRecorder{
		ResponseWriter: w,
	}
	defer func() {
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, recorder.Status, time.Since(start))
	}()
	l.handler.ServeHTTP(recorder, r)
}
