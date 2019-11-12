package storage

import (
	"log"

	"github.com/Charana123/torrent/go-torrent/torrent"

	"github.com/boljen/go-bitmap"

	"github.com/spf13/afero"
)

var appFS = afero.NewOsFs()
var openFile = appFS.OpenFile

type Storage interface {
	Init(tor *torrent.Torrent)
	BlockReadRequest(pieceIndex, blockByteOffset, length int) (blockData []byte, err error)
	WritePieceRequest(pieceIndex int, data []byte) (err error)
	GetCurrentDownloadState() (clientBitfield bitmap.Bitmap, completed bool, left int)
}

func fail(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
