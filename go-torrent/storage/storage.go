package storage

import (
	"log"

	"github.com/boljen/go-bitmap"

	"github.com/spf13/afero"
)

var appFS = afero.NewOsFs()
var openFile = appFS.OpenFile

type Storage interface {
	Init()
	BlockReadRequest(pieceIndex, blockByteOffset, length int) (blockData []byte, err error)
	WritePieceRequest(pieceIndex int, data []byte) (err error)
	GetCurrentDownloadState() (clientBitfield bitmap.Bitmap, completed bool, left int)
}

func fail(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
