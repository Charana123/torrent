package piece

import (
	"sort"

	"github.com/Charana123/torrent/go-torrent/storage"

	"github.com/Charana123/torrent/go-torrent/torrent"
	"github.com/Charana123/torrent/go-torrent/wire"
	bitmap "github.com/boljen/go-bitmap"
)

type sequential struct {
	*rarestFirst
	currPiece int
	currBlock int
}

func NewSequentialPieceManager(
	tor *torrent.Torrent,
	storage storage.Storage,
	clientBitField bitmap.Bitmap) PieceManager {

	rarestFirstt := NewRarestFirstPieceManager(tor, storage, clientBitField)
	seq := &sequential{
		rarestFirst: rarestFirstt.(*rarestFirst),
		currPiece:   0,
		currBlock:   0,
	}

	return seq
}

// ask the storage api to map the file size to which piece ? get pieces per file, map byte offset to piece and block, set and start downloading
func (pm *sequential) SendBlockRequests(id string, wire wire.Wire, peerBitfield *bitmap.Bitmap) error {
	pm.Lock()
	defer pm.Unlock()

	var pieceIndex int
	var blocks int

	// send block requests for the following pieces
	if pi, ok := pm.peerToPiece[id]; ok {
		// If the peer is downloading a certain piece, check if it is currently required
		// i.e. check if the index is after the current index
		// problem - what is the user moved backwards ?
		// check if peer doesn't have any previous pieces upto the seq piece
		// otherwise, help the other peers to download that piece
		// only if it is
		pieceIndex = pi
		blocks = 1
	} else {
		// Otherwise find what piece the peer can help download next, download blocks for that piece
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
			if pieceIndex == pm.tor.NumPieces-1 && blockIndex == pm.numBlockInLastPiece-1 {
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
