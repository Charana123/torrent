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

type Disk interface {
	BlockReadRequest(pieceIndex, blockByteOffset, length int, resp chan *blockReadResponse)
	WritePieceRequest(pieceIndex int, data []byte)
}

type disk struct {
	metainfo  *metaInfo
	files     []afero.File
	fileLocks []*sync.Mutex
}

func NewDisk(
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

func (d *disk) readBlock(fileIndex, offset, length int) ([]byte, error) {

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
		if err != nil {
			return nil, err
		}

		binary.Write(blockData, binary.BigEndian, data)
		length = length - (d.metainfo.Info.Files[fileIndex].Length - offset)
		offset = 0
		fileIndex++
	}
	return blockData.Bytes(), nil
}

func (d *disk) BlockReadRequest(pieceIndex, blockByteOffset, length int) ([]byte, error) {
	offset := pieceIndex*d.metainfo.Info.PieceLength + blockByteOffset
	var err error
	var block []byte
	if len(d.metainfo.Info.Files) > 0 {
		// Multiple File Mode
		for fileIndex := 0; fileIndex < len(d.metainfo.Info.Files); fileIndex++ {
			if offset >= d.metainfo.Info.Files[fileIndex].Length-1 {
				offset -= d.metainfo.Info.Files[fileIndex].Length
			} else {
				block, err := d.readBlock(fileIndex, offset, length)
				break
			}
		}
	} else {
		// Single File Mode
		block, err := d.readBlock(0, offset, length)
	}
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (d *disk) writePiece(fileIndex, offset int, data []byte) error {

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
		if err != nil {
			return err
		}

		data = data[writeLen:]
		offset = 0
		fileIndex++
	}
	return nil
}

func (d *disk) WritePieceRequest(pieceIndex int, data []byte) error {
	offset := pieceIndex * d.metainfo.Info.PieceLength
	var err error
	if len(d.metainfo.Info.Files) > 0 {
		// Multiple File Mode
		for fileIndex := 0; fileIndex < len(d.metainfo.Info.Files); fileIndex++ {
			if offset >= d.metainfo.Info.Files[fileIndex].Length-1 {
				offset -= d.metainfo.Info.Files[fileIndex].Length
			} else {
				err = d.writePiece(fileIndex, offset, data)
			}
		}
	} else {
		// Single File Mode
		err = d.writePiece(0, offset, data)
	}
	if err != nil {
		return err
	}
	return nil
}
