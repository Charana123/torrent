package torrent

import "io"

type Torrent struct {
}

func (t *Torrent) GetStream() Reader {

}

type Reader interface {
	io.Reader
	io.Seeker
	io.Closer
}

type TorrentStream struct {
}

func (r Reader) Read(b []byte) (n int, err error) {

}
