package client

import (
	"io"

	"github.com/Charana123/torrent/go-torrent/storage"
)

type torrentReadSeeker struct {
	io.ReadSeeker
	s *storage.Storage
}

// a file
func NewTorrentReadSeeker() {
	// get the file index and the file offset, read the block
	// if piece
}
