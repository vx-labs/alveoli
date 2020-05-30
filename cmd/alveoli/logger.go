package main

import (
	"log"
	"net/http"
	"time"
)

type Logger struct {
	handler http.Handler
}

func (l *Logger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	}()
	l.handler.ServeHTTP(w, r)
}
