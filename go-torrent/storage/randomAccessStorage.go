package storage

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync"

	underscore "github.com/ahl5esoft/golang-underscore"

	"github.com/Charana123/torrent/go-torrent/torrent"
	"github.com/boljen/go-bitmap"
	"github.com/spf13/afero"
)

type randomAccessStorage struct {
	sync.RWMutex
	torrent       *torrent.Torrent
	fileLocks     []*sync.Mutex
	files         []afero.File
	fileOffsets   []int
	dataDirectory string
	rootDirectory string
}

func NewRandomAccessStorage(dataDirectory string) Storage {
	return &randomAccessStorage{
		dataDirectory: dataDirectory,
	}
}

func openOrCreateFile(path string, length int) afero.File {
	file, err := openFile(path, os.O_CREATE|os.O_RDWR, 0755)
	fail(err)
	err = file.Truncate(int64(length))
	fail(err)
	return file
}

func (d *randomAccessStorage) Init(tor *torrent.Torrent) {
	d.Lock()
	defer d.Unlock()

	d.torrent = tor
	infoHashHex := hex.EncodeToString(d.torrent.InfoHash)
	d.rootDirectory = strings.Join([]string{d.dataDirectory, infoHashHex}, "/")

	if len(d.torrent.MetaInfo.Info.Files) > 0 {
		// Multiple File Mode

		// Create root directory
		rootDirectory := strings.Join([]string{d.rootDirectory, d.torrent.MetaInfo.Info.Name}, "/")
		if _, err := appFS.Stat(rootDirectory); os.IsNotExist(err) {
			err := appFS.MkdirAll(rootDirectory, 0755)
			fail(err)
		}

		// Create sub-directories and create/open file handlers
		offset := 0
		for _, file := range d.torrent.MetaInfo.Info.Files {
			subdir := strings.Join(append([]string{rootDirectory}, file.Path[:len(file.Path)-1]...), "/")
			if _, err := appFS.Stat(subdir); os.IsNotExist(err) {
				err := appFS.MkdirAll(subdir, 0755)
				fail(err)
			}
			path := strings.Join(append([]string{rootDirectory}, file.Path...), "/")
			d.files = append(d.files, openOrCreateFile(path, file.Length))
			d.fileLocks = append(d.fileLocks, &sync.Mutex{})
			d.fileOffsets = append(d.fileOffsets, offset)
			offset += file.Length
		}
	} else {
		// Create root directory
		if _, err := appFS.Stat(d.rootDirectory); os.IsNotExist(err) {
			err := appFS.Mkdir(d.rootDirectory, 0755)
			fail(err)
		}
		// Single File Mode
		fileName := strings.Join([]string{d.rootDirectory, d.torrent.MetaInfo.Info.Name}, "/")
		file := openOrCreateFile(fileName, d.torrent.MetaInfo.Info.Length)
		d.files = append(d.files, file)
		d.fileLocks = append(d.fileLocks, &sync.Mutex{})
		d.fileOffsets = append(d.fileOffsets, 0)

		d.torrent.MetaInfo.Info.Files = append(d.torrent.MetaInfo.Info.Files, torrent.File{
			Length: d.torrent.MetaInfo.Info.Length,
			Path:   []string{d.torrent.MetaInfo.Info.Name},
		})
	}
}

func (d *randomAccessStorage) find(globalOffset int) (int, int, error) {
	i := 0
	j := len(d.files)
	for i < j {
		fileIndex := (i + j) / 2
		if globalOffset >= d.fileOffsets[fileIndex] &&
			globalOffset < d.fileOffsets[fileIndex]+d.torrent.MetaInfo.Info.Files[fileIndex].Length {
			fileOffset := globalOffset - d.fileOffsets[fileIndex]
			return fileIndex, fileOffset, nil
		}
		if globalOffset >= d.fileOffsets[fileIndex] {
			i = fileIndex + 1
		} else {
			j = fileIndex
		}
	}
	return 0, 0, fmt.Errorf("File doesn't exist")
}

func min(i, j int) int {
	if i < j {
		return i
	}
	return j
}

func (d *randomAccessStorage) readBlock(fileIndex, fileOffset, blockLength int) ([]byte, error) {

	blockData := &bytes.Buffer{}
	for i := 0; i < 2 && blockLength > 0; i++ {
		length := min(d.torrent.MetaInfo.Info.Files[fileIndex].Length-fileOffset, blockLength)
		data := make([]byte, length)

		d.fileLocks[fileIndex].Lock()
		_, err := d.files[fileIndex].ReadAt(data, int64(fileOffset))
		d.fileLocks[fileIndex].Unlock()
		fail(err)
		binary.Write(blockData, binary.BigEndian, data)

		blockLength -= length
		fileIndex++
		if blockLength > 0 && fileIndex >= len(d.files) {
			return ([]byte)(nil), fmt.Errorf("reading beyond end of last file")
		}
		fileOffset = 0
	}
	return blockData.Bytes(), nil
}

func (d *randomAccessStorage) BlockReadRequest(pieceIndex, blockByteOffset, blockLength int) ([]byte, error) {
	// Generic checks
	if pieceIndex < 0 || pieceIndex >= d.torrent.NumPieces {
		return ([]byte)(nil), fmt.Errorf("Invalid piece index")
	}
	if blockByteOffset > d.torrent.MetaInfo.Info.PieceLength {
		return ([]byte)(nil), fmt.Errorf("begin (byte offset within piece) larger than piece")
	}
	if blockLength > d.torrent.MetaInfo.Info.PieceLength {
		return ([]byte)(nil), fmt.Errorf("block size cannot be larger than piece size")
	}

	globalOffset := pieceIndex*d.torrent.MetaInfo.Info.PieceLength + blockByteOffset
	fileIndex, fileOffset, err := d.find(globalOffset)
	fail(err)
	block, err := d.readBlock(fileIndex, fileOffset, blockLength)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (d *randomAccessStorage) writePiece(fileIndex, fileOffset int, data []byte) error {

	for i := 0; i < 2 && len(data) > 0; i++ {
		length := min(d.torrent.MetaInfo.Info.Files[fileIndex].Length-fileOffset, len(data))
		d.fileLocks[fileIndex].Lock()
		d.files[fileIndex].WriteAt(data[:length], int64(fileOffset))
		d.fileLocks[fileIndex].Unlock()

		// after writing, check
		data = data[length:]
		fileIndex++
		if len(data) > 0 && fileIndex >= len(d.files) {
			fmt.Println("fileIndex: ", fileIndex)
			return fmt.Errorf("writing beyond end of last file")
		}
		fileOffset = 0
	}
	return nil
}

func (d *randomAccessStorage) WritePieceRequest(pieceIndex int, data []byte) error {
	globalOffset := pieceIndex * d.torrent.MetaInfo.Info.PieceLength
	fileIndex, fileOffset, err := d.find(globalOffset)
	fail(err)
	err = d.writePiece(fileIndex, fileOffset, data)
	if err != nil {
		return err
	}
	return nil
}

func (d *randomAccessStorage) GetCurrentDownloadState() (bitmap.Bitmap, bool, int) {
	clientBitfield := bitmap.New(d.torrent.NumPieces)
	// read pieces sequentially, validating the checksums
	for pieceIndex := 0; pieceIndex < d.torrent.NumPieces; pieceIndex++ {
		var piece []byte
		var err error
		if pieceIndex == d.torrent.NumPieces-1 {
			bytesInLastPiece := d.torrent.Length - ((d.torrent.NumPieces - 1) * d.torrent.MetaInfo.Info.PieceLength)
			piece, err = d.BlockReadRequest(pieceIndex, 0, bytesInLastPiece)
		} else {
			piece, err = d.BlockReadRequest(pieceIndex, 0, d.torrent.MetaInfo.Info.PieceLength)
		}
		fail(err)
		expectedChecksum := []byte(d.torrent.MetaInfo.Info.Pieces)[pieceIndex*20 : (pieceIndex+1)*20]
		actualChecksum := sha1.Sum(piece)
		if bytes.Equal(expectedChecksum, actualChecksum[:]) {
			clientBitfield.Set(pieceIndex, true)
		}
	}

	piecesDownloaded := 0
	underscore.
		Chain(clientBitfield.Data(false)).
		Distinct(func(b byte) bool { return b == 1 }).
		Value(&piecesDownloaded)
	left := d.torrent.Length - piecesDownloaded*d.torrent.MetaInfo.Info.PieceLength
	completed := false
	if piecesDownloaded == d.torrent.NumPieces {
		completed = true
	}

	return clientBitfield, completed, left
}
