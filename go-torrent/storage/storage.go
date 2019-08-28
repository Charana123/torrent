package storage

import (
	"log"

	"github.com/boljen/go-bitmap"

	"github.com/spf13/afero"
)

var appFS = afero.NewOsFs()
var openFile = appFS.OpenFile

type Storage interface {
	BlockReadRequest(pieceIndex, blockByteOffset, length int) (blockData []byte, err error)
	WritePieceRequest(pieceIndex int, data []byte) (err error)
	GetDownloadState() (clientBitfield bitmap.Bitmap, completed bool)
}

func fail(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
