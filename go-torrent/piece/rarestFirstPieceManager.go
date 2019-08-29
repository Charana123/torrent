package piece

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
	"sync"

	"github.com/Charana123/torrent/go-torrent/torrent"
	"github.com/Charana123/torrent/go-torrent/wire"
	bitmap "github.com/boljen/go-bitmap"
	mapset "github.com/deckarep/golang-set"
)

type rarestFirst struct {
	sync.RWMutex
	clientBitField bitmap.Bitmap
	tor            *torrent.Torrent
	numBlocks      int
	peerToPiece    map[string]int
	pieceInfo      []*pieceInfo
}

type pieceInfo struct {
	downloaded  bool
	downloading bool
	blocks      []*blockInfo
	availabilty int
	peers       mapset.Set
}

type blockInfo struct {
	downloaded  bool
	downloading bool
	data        []byte
}

func NewRarestFirstPieceManager(
	tor *torrent.Torrent,
	clientBitField bitmap.Bitmap) PieceManager {

	pm := &rarestFirst{
		clientBitField: clientBitField,
		tor:            tor,
		numBlocks:      tor.MetaInfo.Info.PieceLength / BLOCK_SIZE,
		peerToPiece:    make(map[string]int),
	}

	pis := make([]*pieceInfo, 0)
	for i := 0; i < pm.tor.NumPieces; i++ {
		pi := &pieceInfo{}
		pi.blocks = make([]*blockInfo, 0)
		for j := 0; j < pm.numBlocks; j++ {
			pi.blocks = append(pi.blocks, &blockInfo{})
		}
		pi.peers = mapset.NewSet()
		pis = append(pis, pi)
	}
	pm.pieceInfo = pis

	return pm
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
		for _, block := range pm.pieceInfo[pieceIndex].blocks {
			block.downloading = false
		}
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
		for _, block := range pm.pieceInfo[pieceIndex].blocks {
			block.downloading = false
		}
		delete(pm.peerToPiece, id)
	}
}

func (pm *rarestFirst) PieceHave(id string, pieceIndex int) {
	pm.Lock()
	defer pm.Unlock()

	pm.pieceInfo[pieceIndex].availabilty++
}

func (pm *rarestFirst) WriteBlock(id string, pieceIndex, blockIndex int, data []byte) (bool, []byte, mapset.Set, error) {
	pm.Lock()
	defer pm.Unlock()

	// Check pieceIndex and blockIndex and set block as downloaded
	if pi, ok := pm.peerToPiece[id]; !ok || pi != pieceIndex {
		return false, ([]byte)(nil), (mapset.Set)(nil), fmt.Errorf("downloaded block from incorrent piece")
	}
	if !pm.pieceInfo[pieceIndex].blocks[blockIndex].downloading {
		return false, ([]byte)(nil), (mapset.Set)(nil), fmt.Errorf("downloaded incorrent block")
	}
	if len(data) != BLOCK_SIZE {
		return false, ([]byte)(nil), (mapset.Set)(nil), fmt.Errorf("incorrent block size")
	}
	pm.pieceInfo[pieceIndex].blocks[blockIndex].downloaded = true
	pm.pieceInfo[pieceIndex].blocks[blockIndex].downloading = false
	pm.pieceInfo[pieceIndex].blocks[blockIndex].data = data
	pm.pieceInfo[pieceIndex].peers.Add(id)

	// If all blocks for piece are downloaded, set piece as downloaded
	for i := len(pm.pieceInfo[pieceIndex].blocks) - 1; i >= 0; i-- {
		block := pm.pieceInfo[pieceIndex].blocks[i]
		if !block.downloaded {
			return false, ([]byte)(nil), (mapset.Set)(nil), nil
		}
	}

	// Write piece to disk
	pm.pieceInfo[pieceIndex].downloaded = true
	pm.pieceInfo[pieceIndex].downloading = false
	delete(pm.peerToPiece, id)
	pm.clientBitField.Set(pieceIndex, true)

	// Check piece's checksum
	piece := &bytes.Buffer{}
	for _, block := range pm.pieceInfo[pieceIndex].blocks {
		binary.Write(piece, binary.BigEndian, block.data)
	}
	return false, piece.Bytes(), pm.pieceInfo[pieceIndex].peers, nil
}

func (pm *rarestFirst) SendBlockRequests(id string, wire wire.Wire, peerBitfield *bitmap.Bitmap) error {
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
		pieces := make([]int, 0)
		for pieceIndex := 0; pieceIndex < peerBitfield.Len(); pieceIndex++ {
			if peerBitfield.Get(pieceIndex) {
				if !pm.pieceInfo[pieceIndex].downloaded && !pm.pieceInfo[pieceIndex].downloading {
					pieces = append(pieces, pieceIndex)
				}
			}
		}
		if len(pieces) == 0 {
			return wire.SendUnInterested()
		}
		// sort them by rarity
		sort.Slice(pieces, func(i, j int) bool {
			p1, p2 := pieces[i], pieces[j]
			return pm.pieceInfo[p1].availabilty < pm.pieceInfo[p2].availabilty
		})

		pieceIndex = pieces[0]
		blocks = MAX_OUTSTANDING_REQUESTS
		pm.peerToPiece[id] = pieceIndex
		pm.pieceInfo[pieceIndex].downloading = true
	}

	for blockIndex, block := range pm.pieceInfo[pieceIndex].blocks {
		if !block.downloaded && !block.downloading {
			err := wire.SendRequest(pieceIndex, blockIndex, BLOCK_SIZE)
			if err != nil {
				return err
			}
			pm.pieceInfo[pieceIndex].blocks[blockIndex].downloading = true
			blocks--
			if blocks == 0 {
				return nil
			}
		}
	}
	return nil
}
