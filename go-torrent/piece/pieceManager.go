package piece

import (
	"github.com/Charana123/torrent/go-torrent/wire"
	bitmap "github.com/boljen/go-bitmap"
)

var (
	MAX_OUTSTANDING_REQUESTS = 5
	BLOCK_SIZE               = 16384 // 2^14
)

type PieceManager interface {
	GetBitField() []byte
	PeerChoked(id string)
	PeerStopped(id string, peerBitfield *bitmap.Bitmap)
	PieceHave(id string, pieceIndex int)
	WriteBlock(id string, pieceIndex, blockIndex int, data []byte) error
	SendBlockRequests(id string, wire wire.Wire, peerBitfield *bitmap.Bitmap)
}
