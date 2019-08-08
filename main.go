package main

import (
    "log"
    "net/http"
)

func main() {
	staticServeHandler := http.FileServer(http.Dir("/Users/charana/Documents/go/src/github.com/Charana123/torrent/public/"))
	http.Handle("/public/", http.StripPrefix("/public/", staticServeHandler))
	http.ListenAndServe(":8080", logRequest(http.DefaultServeMux))
}

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s\n", r.RemoteAddr, r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}
