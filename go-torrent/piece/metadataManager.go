package piece

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"

	"github.com/jackpal/bencode-go"

	"github.com/Charana123/torrent/go-torrent/torrent"
	"github.com/Charana123/torrent/go-torrent/wire"
)

const (
	METADATA_PIECE_SIZE = 16384 // 16 KiB
)

type MetadataManager interface {
	Init(metadataSize int)
	SendPieceRequest(id string, wire wire.Wire) (err error)
	WritePiece(pieceIndex int, piece []byte) (downloadComplete bool)
	GetNumMetaPieces() int
}

type metadataManager struct {
	muri             *torrent.MagnetURI
	metadata         []byte
	metaPieceInfo    []*MetaPieceInfo
	numMetaPieces    int
	piecesDownloaded int
	dowloadedChan    chan *torrent.Torrent
}

type MetaPieceInfo struct {
	downloaded  bool
	downloading bool
}

func NewMetadataManager(muri *torrent.MagnetURI) (MetadataManager, chan *torrent.Torrent) {
	mm := &metadataManager{
		muri: muri,
	}
	mm.dowloadedChan = make(chan *torrent.Torrent)
	return mm, mm.dowloadedChan
}

func (mdMgr *metadataManager) Init(metadataSize int) {
	mdMgr.numMetaPieces = metadataSize/METADATA_PIECE_SIZE +
		map[bool]int{
			true:  1,
			false: 0,
		}[metadataSize%METADATA_PIECE_SIZE == 0]
	mdMgr.metaPieceInfo = make([]*MetaPieceInfo, mdMgr.numMetaPieces)
	mdMgr.metadata = make([]byte, metadataSize)
}

func (mdMgr *metadataManager) GetNumMetaPieces() int {
	return mdMgr.numMetaPieces
}

func (mdMgr *metadataManager) SendPieceRequest(id string, wire wire.Wire) error {
	var i int = 0
	for ; i < mdMgr.numMetaPieces; i++ {
		if !mdMgr.metaPieceInfo[i].downloaded && !mdMgr.metaPieceInfo[i].downloading {
			err := wire.SendExtendedMetadataRequest(i)
			if err != nil {
				mdMgr.metaPieceInfo[i].downloading = true
			}
			return err
		}
	}
	return nil
}

func (mdMgr *metadataManager) WritePiece(pieceIndex int, piece []byte) bool {
	mdMgr.metaPieceInfo[pieceIndex].downloaded = true
	mdMgr.metaPieceInfo[pieceIndex].downloading = false
	mdMgr.piecesDownloaded++
	copy(mdMgr.metadata[pieceIndex*METADATA_PIECE_SIZE:], piece)
	if mdMgr.piecesDownloaded == mdMgr.numMetaPieces {
		infoHash, _ := hex.DecodeString(mdMgr.muri.InfoHashHex)
		metadataHash := sha1.Sum(mdMgr.metadata)
		if bytes.Equal(metadataHash[:], infoHash) {
			info := &torrent.Info{}
			bencode.Unmarshal(bytes.NewBuffer(mdMgr.metadata), info)
			tor := torrent.NewTorrentFromMagnetURI(mdMgr.muri, info)
			mdMgr.dowloadedChan <- tor
		} else {
			fmt.Println("metadata verification failed")
			// reset!
		}
	}
	return mdMgr.piecesDownloaded == mdMgr.numMetaPieces
}
