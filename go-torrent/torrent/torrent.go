package torrent

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math/rand"

	bencode "github.com/jackpal/bencode-go"
)

var (
	PEER_ID = make([]byte, 20, 20)
)

func init() {
	copy(PEER_ID[:8], []byte("-GT0001-"))
	_, err := rand.Read(PEER_ID[8:])
	if err != nil {
		log.Fatalln(err)
	}
}

type Torrent struct {
	Length    int
	MetaInfo  MetaInfo
	InfoHash  []byte
	NumPieces int
}

type MetaInfo struct {
	Info         Info
	Announce     string
	AnnounceList [][]string `bencode:"announce-list"`
	CreationDate int        `bencode:"creation date"`
	Comment      string
	CreatedBy    string `bencode:"created by"`
	Encoding     string
}

type Info struct {
	PieceLength int `bencode:"piece length"`
	Pieces      string
	Private     int
	Name        string
	Length      int
	Md5sum      string
	Files       []File
}

type File struct {
	Length int
	Md5sum string
	Path   []string
}

func NewTorrentFromMagnetURI(muri *MagnetURI, info *Info) *Torrent {
	infoHash, _ := hex.DecodeString(muri.InfoHashHex)
	tor := &Torrent{
		InfoHash: infoHash,
		MetaInfo: MetaInfo{
			Info:         *info,
			AnnounceList: [][]string{muri.Trackers},
		},
	}
	tor.NumPieces = len(tor.MetaInfo.Info.Pieces) / 20
	for i := 0; i < len(tor.MetaInfo.Info.Files); i++ {
		tor.Length += tor.MetaInfo.Info.Files[i].Length
	}
	return tor
}

func NewTorrent(torrentReader io.ReadSeeker) (*Torrent, error) {
	tor := &Torrent{}

	metaInfo, err := bencode.Decode(torrentReader)
	if err != nil {
		return nil, err
	}
	metaInfoMap, ok := metaInfo.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Malformed torrent file")
	}
	infoMap, ok := metaInfoMap["info"]
	if !ok {
		return nil, fmt.Errorf("Malformed torrent file")
	}

	infoBencode := &bytes.Buffer{}
	bencode.Marshal(infoBencode, infoMap)
	infoHash := sha1.Sum(infoBencode.Bytes())
	tor.InfoHash = infoHash[:]

	torrentReader.Seek(0, 0)
	err = bencode.Unmarshal(torrentReader, &tor.MetaInfo)
	if err != nil {
		return nil, err
	}
	tor.NumPieces = len(tor.MetaInfo.Info.Pieces) / 20

	// Total size of all files
	if len(tor.MetaInfo.Info.Files) > 0 {
		for i := 0; i < len(tor.MetaInfo.Info.Files); i++ {
			tor.Length += tor.MetaInfo.Info.Files[i].Length
		}
	} else {
		tor.Length += tor.MetaInfo.Info.Length
	}
	return tor, nil
}
