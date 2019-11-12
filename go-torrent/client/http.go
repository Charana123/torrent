package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type HTTPServeMux struct {
	*http.ServeMux
	client Client
}

func (sm *HTTPServeMux) uploadTorrent(rw http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		torrentBuff := &bytes.Buffer{}
		torrentBuff.ReadFrom(r.Body)
		torrentReader := bytes.NewReader(torrentBuff.Bytes())
		td := sm.client.AddTorrent(torrentReader)
		td.Start()

		rw.WriteHeader(http.StatusOK)
	} else {
		rw.WriteHeader(http.StatusNotFound)
	}
}

func (sm *HTTPServeMux) magnetTorrent(rw http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if magnetURI := r.URL.Query().Get("uri"); len(magnetURI) > 0 {
			fmt.Println("magnetURI: ", magnetURI)
			td, err := sm.client.AddMagnet(magnetURI)
			if err != nil {
				rw.WriteHeader(http.StatusBadRequest)
			}
			td.Start()
		} else {
			rw.WriteHeader(http.StatusBadRequest)
		}
		rw.WriteHeader(http.StatusOK)
	} else {
		rw.WriteHeader(http.StatusNotFound)
	}
}

func (sm *HTTPServeMux) commandTorrent(rw http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		jsonMap := make(map[string]interface{})
		json.NewDecoder(r.Body).Decode(jsonMap)
		_, ok1 := jsonMap["torrentID"]
		command, ok2 := jsonMap["command"]
		if !ok1 || !ok2 {
			rw.WriteHeader(http.StatusBadRequest)
		}
		_, ok3 := jsonMap["fileIndex"]
		switch command {
		case "START":
			if !ok3 {
				// sm.client.StartTorrent(torrentID.(string))
			} else {
				// sm.client.StartFile(torrentID.(string), fileIndex.(int))
			}
		case "STOP":
			if !ok3 {
				// sm.client.StopTorrent(torrentID.(string))
			} else {
				// sm.client.StopFile(torrentID.(string), fileIndex.(int))
			}
		case "VERIFY":
		}
	} else {
		rw.WriteHeader(http.StatusBadRequest)
	}
	return
}

type clientData struct {
}

func (sm *HTTPServeMux) getStats(rw http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		clientData := &clientData{}
		data, _ := json.Marshal(clientData)
		rw.Write(data)
		rw.WriteHeader(http.StatusFound)
	} else {
		rw.WriteHeader(http.StatusBadRequest)
	}
}

func (sm *HTTPServeMux) streamTorrent(rw http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		jsonMap := make(map[string]interface{})
		json.NewDecoder(r.Body).Decode(jsonMap)
		_, ok1 := jsonMap["torrentID"]
		_, ok2 := jsonMap["fileIndex"]
		if !ok1 || !ok2 {
			rw.WriteHeader(http.StatusBadRequest)
		}
	}
	return
}

func NewHTTPServeMux(storagePath string) *HTTPServeMux {
	sm := http.NewServeMux()
	client := NewClient(storagePath)
	httpSM := &HTTPServeMux{
		ServeMux: sm,
		client:   client,
	}
	httpSM.HandleFunc("/upload", httpSM.uploadTorrent)
	httpSM.HandleFunc("/magnet", httpSM.magnetTorrent)
	httpSM.HandleFunc("/command", httpSM.commandTorrent)
	httpSM.HandleFunc("/stream", httpSM.streamTorrent)
	return httpSM
}
