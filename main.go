package main

import (
	"time"

	"github.com/Charana123/torrent/go-torrent/download"
)

func main() {
	d := download.NewDownload()
	err := d.Start("/Users/charana/Downloads/malone.torrent")
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Hour)
}
