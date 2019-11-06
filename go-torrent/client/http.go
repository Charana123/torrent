package client

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type HTTPServeMux struct {
	*http.ServeMux
	client Client
}

func (sm *HTTPServeMux) uploadTorrent(rw http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Add torrent to client
		torrentBuff := &bytes.Buffer{}
		torrentBuff.ReadFrom(r.Body)
		torrentReader := bytes.NewReader(torrentBuff.Bytes())
		sm.client.AddTorrent(torrentReader)

		rw.WriteHeader(http.StatusOK)
	} else {
		rw.WriteHeader(http.StatusNotFound)
	}
}

func (sm *HTTPServeMux) commandTorrent(rw http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		jsonMap := make(map[string]interface{})
		json.NewDecoder(r.Body).Decode(jsonMap)
		torrentID, ok1 := jsonMap["torrentID"]
		command, ok2 := jsonMap["command"]
		if !ok1 || !ok2 {
			rw.WriteHeader(http.StatusBadRequest)
		}
		fileIndex, ok3 := jsonMap["fileIndex"]
		switch command {
		case "START":
			if !ok3 {
				sm.client.StartTorrent(torrentID.(string))
			} else {
				sm.client.StartFile(torrentID.(string), fileIndex.(int))
			}
		case "STOP":
			if !ok3 {
				sm.client.StopTorrent(torrentID.(string))
			} else {
				sm.client.StopFile(torrentID.(string), fileIndex.(int))
			}
		case "VERIFY":
		}
	}
	return
}

func (sm *HTTPServeMux) stream(rw http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		jsonMap := make(map[string]interface{})
		json.NewDecoder(r.Body).Decode(jsonMap)
		torrentID, ok1 := jsonMap["torrentID"]
		fileIndex, ok2 := jsonMap["fileIndex"]
		if !ok1 || !ok2 {
			rw.WriteHeader(http.StatusBadRequest)
		}

	}
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
