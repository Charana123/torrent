package piece

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
	"sync"

	"github.com/Charana123/torrent/go-torrent/disk"
	"github.com/Charana123/torrent/go-torrent/torrent"
	"github.com/Charana123/torrent/go-torrent/wire"
	bitmap "github.com/boljen/go-bitmap"
)

type rarestFirst struct {
	sync.RWMutex
	clientBitField bitmap.Bitmap
	numPieces      int
	numBlocks      int
	disk           disk.Disk
	peerToPiece    map[string]int
	pieceInfo      []*pieceInfo
}

type pieceInfo struct {
	downloaded  bool
	downloading bool
	blocks      []*blockInfo
	availabilty int
	peers       []string
}

type blockInfo struct {
	downloaded  bool
	downloading bool
	data        []byte
}

func (pm *rarestFirst) GetBitField() []byte {
	pm.RLock()
	defer pm.RUnlock()

	return pm.clientBitField.Data(true)
}

func (pm *rarestFirst) PeerChoked(id string) {
	pm.Lock()
	defer pm.Unlock()

	if pieceIndex, ok := pm.peerToPiece[id]; ok {
		pm.pieceInfo[pieceIndex].downloading = false
		delete(pm.peerToPiece, id)
	}
}

func (pm *rarestFirst) PeerStopped(id string, peerBitfield *bitmap.Bitmap) {
	pm.Lock()
	defer pm.Unlock()

	// Update piece availabilities
	for pieceIndex := 0; pieceIndex < peerBitfield.Len(); pieceIndex++ {
		if peerBitfield.Get(pieceIndex) {
			pm.pieceInfo[pieceIndex].availabilty--
		}
	}

	if pieceIndex, ok := pm.peerToPiece[id]; ok {
		pm.pieceInfo[pieceIndex].downloading = false
		delete(pm.peerToPiece, id)
	}
}

func (pm *rarestFirst) PieceHave(id string, pieceIndex int) {
	pm.Lock()
	defer pm.Unlock()

	pm.pieceInfo[pieceIndex].availabilty++
}

func (pm *rarestFirst) WriteBlock(id string, pieceIndex, blockIndex int, data []byte) error {
	pm.Lock()
	defer pm.Unlock()

	// TODO: add more checks
	// Check pieceIndex and blockIndex and set block as downloaded
	if pi, ok := pm.peerToPiece[id]; !ok || pi != pieceIndex {
		return fmt.Errorf("downloaded block from incorrent piece")
	}
	if !pm.pieceInfo[pieceIndex].blocks[blockIndex].downloading {
		return fmt.Errorf("downloaded incorrent block")
	}
	if len(data) != BLOCK_SIZE {
		return fmt.Errorf("incorrent block size")
	}
	pm.pieceInfo[pieceIndex].blocks[blockIndex].downloaded = true
	pm.pieceInfo[pieceIndex].blocks[blockIndex].downloading = false
	pm.pieceInfo[pieceIndex].blocks[blockIndex].data = data
	if id != pm.pieceInfo[pieceIndex].peers[len(pm.pieceInfo[pieceIndex].peers)-1] {
		pm.pieceInfo[pieceIndex].peers = append(pm.pieceInfo[pieceIndex].peers, id)
	}

	// If all blocks for piece are downloaded, set piece as downloaded
	for i := len(pm.pieceInfo[pieceIndex].blocks); i >= 0; i-- {
		block := pm.pieceInfo[pieceIndex].blocks[i]
		if !block.downloaded {
			return nil
		}
	}
	pm.pieceInfo[pieceIndex].downloaded = true
	pm.pieceInfo[pieceIndex].downloading = false
	delete(pm.peerToPiece, id)

	// Write piece to disk
	piece := &bytes.Buffer{}
	for _, block := range pm.pieceInfo[pieceIndex].blocks {
		binary.Write(piece, binary.BigEndian, block.data)
	}
	// check sha1 checksum
	pm.disk.WritePieceRequest(pieceIndex, piece.Bytes())

	return nil
}

func (pm *rarestFirst) SendBlockRequests(id string, wire wire.Wire, peerBitfield *bitmap.Bitmap) {
	pm.Lock()
	defer pm.Unlock()

	var pieceIndex int
	var blocks int

	// Get the piece
	if pi, ok := pm.peerToPiece[id]; ok {
		// If the peer is downloading a certain piece, continue downloading its blocks
		pieceIndex = pi
		blocks = 1
	} else {
		// Find the peer's rarest piece that the client doesn't have
		pieces := []int{}
		for pieceIndex := 0; pieceIndex < peerBitfield.Len(); pieceIndex++ {
			if peerBitfield.Get(pieceIndex) {
				if !pm.pieceInfo[pieceIndex].downloaded && !pm.pieceInfo[pieceIndex].downloading {
					pieces = append(pieces, pieceIndex)
				}
			}
		}
		if len(pieces) == 0 {
			return
		}
		// sort them by rarity
		sort.Slice(pieces, func(i, j int) bool {
			p1, p2 := pieces[i], pieces[j]
			return pm.pieceInfo[p1].availabilty < pm.pieceInfo[p2].availabilty
		})

		pm.peerToPiece[id] = pieceIndex
		pm.pieceInfo[pieceIndex].downloading = true
		pieceIndex = pieces[0]
		blocks = MAX_OUTSTANDING_REQUESTS
	}

	for blockIndex, block := range pm.pieceInfo[pieceIndex].blocks {
		if !block.downloaded && !block.downloading {
			wire.SendRequest(pieceIndex, blockIndex, BLOCK_SIZE)
			pm.pieceInfo[pieceIndex].blocks[blockIndex].downloading = true
			blocks--
			if blocks < 0 {
				return
			}
		}
	}
}

func NewRarestFirstPieceManager(
	t *torrent.Torrent,
	disk disk.Disk) PieceManager {

	pm := &rarestFirst{
		disk:        disk,
		numPieces:   t.NumPieces,
		numBlocks:   t.MetaInfo.Info.PieceLength / BLOCK_SIZE,
		peerToPiece: make(map[string]int),
	}

	pieceInfo := make([]*pieceInfo, pm.numPieces)
	for _, pi := range pieceInfo {
		pi.blocks = make([]*blockInfo, pm.numBlocks)
	}
	pm.pieceInfo = pieceInfo

	return nil
}
