package main

import (
	"time"

	"github.com/Charana123/torrent/go-torrent/download"
)

func main() {
	d := download.NewDownload()
	err := d.Start("/Users/deepaninandasena/Desktop/Work/go/src/github.com/Charana123/torrent/malone.torrent")
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Hour)
}
