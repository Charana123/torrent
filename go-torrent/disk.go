package torrent

import (
	"bytes"
	"encoding/binary"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/spf13/afero"
)

var appFS = afero.NewOsFs()
var openFile = appFS.OpenFile

type disk struct {
	metainfo  *metaInfo
	files     []afero.File
	fileLocks []*sync.Mutex
}

func newDisk(
	metainfo *metaInfo) *disk {

	disk := &disk{
		metainfo: metainfo,
	}
	return disk
}

func fail(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func openOrCreateFile(path string) afero.File {
	file, err := openFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		log.Fatalln(err)
	}
	return file
}

func (d *disk) init() {
	if len(d.metainfo.Info.Files) > 0 {
		// Multiple File Mode

		// Create root directory
		if _, err := appFS.Stat(d.metainfo.Info.Name); os.IsNotExist(err) {
			err := appFS.Mkdir(d.metainfo.Info.Name, 0755)
			fail(err)
		}

		// Create sub-directories and create/open file handlers
		for _, file := range d.metainfo.Info.Files {
			subdir := strings.Join(append([]string{d.metainfo.Info.Name}, file.Path[:len(file.Path)-1]...), "/")
			if _, err := appFS.Stat(subdir); os.IsNotExist(err) {
				err := appFS.MkdirAll(subdir, 0755)
				fail(err)
			}
			path := strings.Join(append([]string{d.metainfo.Info.Name}, file.Path...), "/")
			d.files = append(d.files, openOrCreateFile(path))
			d.fileLocks = append(d.fileLocks, &sync.Mutex{})
		}

	} else {
		// Single File Mode
		d.files = append(d.files, openOrCreateFile(d.metainfo.Info.Name))
		d.fileLocks = append(d.fileLocks, &sync.Mutex{})
	}
}

func (d *disk) readBlock(fileIndex, offset, length int) []byte {

	blockData := &bytes.Buffer{}
	for length > 0 {
		var data []byte
		if offset+length > d.metainfo.Info.Files[fileIndex].Length {
			data = make([]byte, d.metainfo.Info.Files[fileIndex].Length-offset)
		} else {
			data = make([]byte, length)
		}
		d.fileLocks[fileIndex].Lock()
		_, err := d.files[fileIndex].ReadAt(data, int64(offset))
		d.fileLocks[fileIndex].Unlock()
		fail(err)

		binary.Write(blockData, binary.BigEndian, data)
		length = length - (d.metainfo.Info.Files[fileIndex].Length - offset)
		offset = 0
		fileIndex++
	}
	return blockData.Bytes()
}

func (d *disk) BlockReadRequest(breq *blockReadRequest, resp chan *blockReadResponse) {
	go func() {
		offset := breq.pieceIndex*d.metainfo.Info.PieceLength + breq.blockByteOffset
		bresp := &blockReadResponse{
			pieceIndex:      breq.pieceIndex,
			blockByteOffset: breq.blockByteOffset,
		}
		if len(d.metainfo.Info.Files) > 0 {
			// Multiple File Mode
			for fileIndex := 0; fileIndex < len(d.metainfo.Info.Files); fileIndex++ {
				if offset >= d.metainfo.Info.Files[fileIndex].Length-1 {
					offset -= d.metainfo.Info.Files[fileIndex].Length
				} else {
					bresp.blockData = d.readBlock(fileIndex, offset, breq.length)
					break
				}
			}
		} else {
			// Single File Mode
			bresp.blockData = d.readBlock(0, offset, breq.length)
		}
		resp <- bresp
	}()
}

func (d *disk) writePiece(fileIndex, offset int, data []byte) {

	for len(data) > 0 {
		var writeLen int
		if offset+len(data) > d.metainfo.Info.Files[fileIndex].Length {
			writeLen = d.metainfo.Info.Files[fileIndex].Length - offset
		} else {
			writeLen = len(data)
		}
		d.fileLocks[fileIndex].Lock()
		_, err := d.files[fileIndex].WriteAt(data[:writeLen], int64(offset))
		d.fileLocks[fileIndex].Unlock()
		fail(err)

		data = data[writeLen:]
		offset = 0
		fileIndex++
	}
}

func (d *disk) WritePieceRequest(preq *pieceWriteRequest) {
	go func() {
		offset := preq.pieceIndex * d.metainfo.Info.PieceLength
		if len(d.metainfo.Info.Files) > 0 {
			// Multiple File Mode
			for fileIndex := 0; fileIndex < len(d.metainfo.Info.Files); fileIndex++ {
				if offset >= d.metainfo.Info.Files[fileIndex].Length-1 {
					offset -= d.metainfo.Info.Files[fileIndex].Length
				} else {
					d.writePiece(fileIndex, offset, preq.data)
					break
				}
			}
		} else {
			// Single File Mode
			d.writePiece(0, offset, preq.data)
		}
	}()
}
