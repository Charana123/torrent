package piece

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/Charana123/torrent/go-torrent/storage"

	"github.com/Charana123/torrent/go-torrent/torrent"
	"github.com/Charana123/torrent/go-torrent/wire"
	bitmap "github.com/boljen/go-bitmap"
	mapset "github.com/deckarep/golang-set"
)

type rarestFirst struct {
	sync.RWMutex
	clientBitField       bitmap.Bitmap
	tor                  *torrent.Torrent
	numBlocks            int
	numBlocksInLastPiece int
	lengthOfLastBlock    int
	peerToPiece          map[string]int
	pieceInfo            []*pieceInfo
	storage              storage.Storage
	piecesDownloaded     int
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
	storage storage.Storage) PieceManager {

	pm := &rarestFirst{
		storage:     storage,
		peerToPiece: make(map[string]int),
	}

	return pm
}

func (pm *rarestFirst) Init(tor *torrent.Torrent, clientBitfield bitmap.Bitmap) {

	pm.tor = tor
	pm.clientBitField = clientBitfield
	bytesInLastPiece := pm.tor.Length - ((pm.tor.NumPieces - 1) * pm.tor.MetaInfo.Info.PieceLength)
	pm.numBlocksInLastPiece = int(math.Ceil(float64(bytesInLastPiece) / float64(BLOCK_SIZE)))
	pm.lengthOfLastBlock = bytesInLastPiece - (pm.numBlocksInLastPiece-1)*BLOCK_SIZE
	pm.numBlocks = pm.tor.MetaInfo.Info.PieceLength / BLOCK_SIZE

	pis := make([]*pieceInfo, 0)
	for i := 0; i < pm.tor.NumPieces; i++ {
		pi := &pieceInfo{}
		pi.blocks = make([]*blockInfo, 0)
		if i == pm.tor.NumPieces-1 {
			for j := 0; j < pm.numBlocksInLastPiece; j++ {
				pi.blocks = append(pi.blocks, &blockInfo{})
			}
		} else {
			for j := 0; j < pm.numBlocks; j++ {
				pi.blocks = append(pi.blocks, &blockInfo{})
			}
		}
		pi.peers = mapset.NewSet()
		pis = append(pis, pi)
	}
	pm.pieceInfo = pis

	for i := 0; i < pm.tor.NumPieces; i++ {
		if pm.clientBitField.Get(i) {
			pm.piecesDownloaded++
		}
	}
}

func (pm *rarestFirst) GetPiecesDownloaded() int {
	pm.RLock()
	defer pm.RUnlock()

	return pm.piecesDownloaded
}

func (pm *rarestFirst) VerifyBitField(bitfield bitmap.Bitmap) {
	pm.Lock()
	defer pm.Unlock()

	for i := 0; i < pm.clientBitField.Len(); i++ {
		if pm.clientBitField.Get(i) && !bitfield.Get(i) {
			pm.pieceInfo[i].downloaded = false
			pm.pieceInfo[i].downloading = false
			for bi := 0; bi < len(pm.pieceInfo[i].blocks); bi++ {
				pm.pieceInfo[i].blocks[bi].downloaded = false
				pm.pieceInfo[i].blocks[bi].downloading = false
			}
		}
	}
	pm.clientBitField = bitfield
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
	if peerBitfield != nil {
		for pieceIndex := 0; pieceIndex < peerBitfield.Len(); pieceIndex++ {
			if peerBitfield.Get(pieceIndex) {
				pm.pieceInfo[pieceIndex].availabilty--
			}
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

func (pm *rarestFirst) WriteBlock(id string, pieceIndex, blockIndex int, data []byte) (bool, mapset.Set, error) {
	pm.Lock()
	defer pm.Unlock()

	// Check pieceIndex and blockIndex and set block as downloaded
	if pi, ok := pm.peerToPiece[id]; !ok || pi != pieceIndex {
		return false, (mapset.Set)(nil), fmt.Errorf("downloaded block from incorrent piece")
	}
	if !pm.pieceInfo[pieceIndex].blocks[blockIndex].downloading {
		return false, (mapset.Set)(nil), fmt.Errorf("downloaded incorrent block")
	}
	if ((pieceIndex != pm.tor.NumPieces-1 || blockIndex != pm.numBlocksInLastPiece-1) && len(data) != BLOCK_SIZE) ||
		((pieceIndex == pm.tor.NumPieces-1 && blockIndex == pm.numBlocksInLastPiece-1) && len(data) != pm.lengthOfLastBlock) {
		return false, (mapset.Set)(nil), fmt.Errorf("incorrent block size")
	}
	pm.pieceInfo[pieceIndex].blocks[blockIndex].downloaded = true
	pm.pieceInfo[pieceIndex].blocks[blockIndex].downloading = false
	pm.pieceInfo[pieceIndex].blocks[blockIndex].data = data
	pm.pieceInfo[pieceIndex].peers.Add(id)

	// If all blocks for piece are downloaded, set piece as downloaded
	for i := 0; i < len(pm.pieceInfo[pieceIndex].blocks); i++ {
		block := pm.pieceInfo[pieceIndex].blocks[i]
		if !block.downloaded {
			return false, (mapset.Set)(nil), nil
		}
	}

	// Check piece's checksum
	piece := &bytes.Buffer{}
	for _, block := range pm.pieceInfo[pieceIndex].blocks {
		binary.Write(piece, binary.BigEndian, block.data)
	}
	pieceData := piece.Bytes()
	expectedChecksum := []byte(pm.tor.MetaInfo.Info.Pieces[20*pieceIndex : 20*(pieceIndex+1)])
	actualChecksum := sha1.Sum(pieceData)
	if !bytes.Equal(expectedChecksum[:], actualChecksum[:]) {
		return true, pm.pieceInfo[pieceIndex].peers, fmt.Errorf("Checksum invalid")
	}

	// Write piece to disk
	err := pm.storage.WritePieceRequest(pieceIndex, pieceData)
	if err != nil {
		return true, nil, err
	}

	// Set piece as downloaded
	pm.pieceInfo[pieceIndex].downloaded = true
	pm.pieceInfo[pieceIndex].downloading = false
	delete(pm.peerToPiece, id)
	pm.clientBitField.Set(pieceIndex, true)
	pm.piecesDownloaded++

	return true, pm.pieceInfo[pieceIndex].peers, nil
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
		// Find the peer's rarest piece that the client doesn't have and isn't
		// being downloaded by another peer
		pieces := make([]int, 0)
		for pieceIndex := 0; pieceIndex < peerBitfield.Len(); pieceIndex++ {
			if peerBitfield.Get(pieceIndex) && !pm.clientBitField.Get(pieceIndex) {
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
			var err error
			if pieceIndex == pm.tor.NumPieces-1 && blockIndex == pm.numBlocksInLastPiece-1 {
				err = wire.SendRequest(pieceIndex, blockIndex*BLOCK_SIZE, pm.lengthOfLastBlock)
			} else {
				err = wire.SendRequest(pieceIndex, blockIndex*BLOCK_SIZE, BLOCK_SIZE)
			}
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
