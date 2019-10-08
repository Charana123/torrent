package piece

import (
	"github.com/Charana123/torrent/go-torrent/wire"
	bitmap "github.com/boljen/go-bitmap"
	mapset "github.com/deckarep/golang-set"
)

var (
	MAX_OUTSTANDING_REQUESTS = 5
	BLOCK_SIZE               = 16384 // 2^14
)

type PieceManager interface {
	GetPiecesDownloaded() (piecesDownloaded int)
	GetBitField() (clientBitfield []byte)
	PeerChoked(id string)
	PeerStopped(id string, peerBitfield *bitmap.Bitmap)
	PieceHave(id string, pieceIndex int)
	WriteBlock(id string, pieceIndex, blockIndex int, data []byte) (downloadedPiece bool, bannedPeers mapset.Set, err error)
	SendBlockRequests(id string, wire wire.Wire, peerBitfield *bitmap.Bitmap) (err error)
}
