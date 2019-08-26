package torrent

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
	"sync"

	bitmap "github.com/boljen/go-bitmap"
)

type rarestFirst struct {
	sync.Mutex
	clientBitField bitmap.Bitmap
	numPieces      int
	numBlocks      int
	disk           Disk
	peerToPiece    map[string]int
	pieceInfo      []*pieceInfo
}

type pieceInfo struct {
	downloaded     bool
	downloading    bool
	blocks         []*blockInfo
	availablePeers int // peer with piece available
}

type blockInfo struct {
	downloaded  bool
	downloading bool
	data        []byte
}

func (pm *rarestFirst) GetBitField() []byte {
	pm.Lock()
	defer pm.Unlock()

	return pm.clientBitField.Data(true)
}

func (pm *rarestFirst) PeerStopped(id string) {
	pm.Lock()
	defer pm.Unlock()

	if pieceIndex, ok := pm.peerToPiece[id]; ok {
		pi := &pieceInfo{}
		pi.blocks = make([]*blockInfo, pm.numBlocks)
		pi.availablePeers = pm.pieceInfo[pieceIndex].availablePeers
		pm.pieceInfo[pieceIndex] = pi
	}
}

func (pm *rarestFirst) PieceHave(id string, pieceIndex int) {
	pm.Lock()
	defer pm.Unlock()

	pm.pieceInfo[pieceIndex].availablePeers++
}

func (pm *rarestFirst) WriteBlock(id string, pieceIndex, blockIndex int, data []byte) error {
	pm.Lock()
	defer pm.Unlock()

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
	pm.disk.WritePieceRequest(pieceIndex, piece.Bytes())

	return nil
}

func (pm *rarestFirst) SendBlockRequests(id string, peer Peer, peerBitfield bitmap.Bitmap) {
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
			return pm.pieceInfo[p1].availablePeers < pm.pieceInfo[p2].availablePeers
		})

		pieceIndex = pieces[0]
		pm.peerToPiece[id] = pieceIndex
		blocks = MAX_OUTSTANDING_REQUESTS
	}

	for blockIndex, block := range pm.pieceInfo[pieceIndex].blocks {
		if !block.downloaded && !block.downloading {
			go peer.SendRequest(pieceIndex, blockIndex, BLOCK_SIZE)
			pm.pieceInfo[pieceIndex].blocks[blockIndex].downloading = true
			blocks--
			if blocks < 0 {
				return
			}
		}
	}
}

func NewRarestFirstPieceManager(t *Torrent, disk Disk) PieceManager {

	pm := &rarestFirst{
		disk:        disk,
		numPieces:   t.numPieces,
		numBlocks:   t.metaInfo.Info.PieceLength / BLOCK_SIZE,
		peerToPiece: make(map[string]int),
	}

	pieceInfo := make([]*pieceInfo, pm.numPieces)
	for _, pi := range pieceInfo {
		pi.blocks = make([]*blockInfo, pm.numBlocks)
	}
	pm.pieceInfo = pieceInfo

	return nil
}
