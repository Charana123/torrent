package main

import (
	"time"

	"github.com/Charana123/torrent/go-torrent/download"
)

func main() {
	d := download.NewDownload()
	err := d.Start("/Users/charana/Downloads/A753EA13F243EF9C4006D103DCBDBC7CABAD8A01.torrent")
	if err != nil {
		panic(err)
	}
	time.Sleep(50 * time.Second)
}
