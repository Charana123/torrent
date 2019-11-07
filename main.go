package main

import (
	"log"
	"net/http"

	"github.com/Charana123/torrent/go-torrent/client"
)

func main() {
	sm := client.NewHTTPServeMux("/Users/charana/Downloads/temp")
	sm.Handle("/", http.FileServer(http.Dir(".")))
	http.ListenAndServe(":8080", logHandler(sm))
}

func logHandler(sm *client.HTTPServeMux) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s\n", r.RemoteAddr, r.Method, r.URL)
		sm.ServeHTTP(rw, r)
	})
}
