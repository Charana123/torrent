package client

import (
	"bytes"
	"net/http"
)

type HTTPServeMux struct {
	*http.ServeMux
	client Client
}

func (sm *HTTPServeMux) upload(rw http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Add torrent
		torrentBuff := &bytes.Buffer{}
		torrentBuff.ReadFrom(r.Body)
		sm.client.AddTorrent(torrentBuff)

		rw.WriteHeader(http.StatusOK)
	} else {
		rw.WriteHeader(http.StatusNotFound)
	}
}

func (sm *HTTPServeMux) start(rw http.ResponseWriter, r *http.Request) {
	return
}

func (sm *HTTPServeMux) stop(rw http.ResponseWriter, r *http.Request) {
	return
}

func (sm *HTTPServeMux) play(rw http.ResponseWriter, r *http.Request) {
	return
}

func NewHTTPServeMux() *HTTPServeMux {
	sm := http.NewServeMux()
	client := NewClient()
	httpSM := &HTTPServeMux{
		ServeMux: sm,
		client:   client,
	}
	httpSM.HandleFunc("/one", httpSM.f)
	return httpSM
}
