package main

import (
	"time"

	"github.com/Charana123/torrent/go-torrent/download"
)

func main() {
	ln, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatal("Port already in use")
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn)
	}

	d := download.NewDownload()
	err := d.Start("/Users/deepaninandasena/Desktop/Work/go/src/github.com/Charana123/torrent/malone.torrent")
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Hour)
}

func 