package torrent

import (
	bitmap "github.com/boljen/go-bitmap"
)

var (
	MAX_OUTSTANDING_REQUESTS = 5
	BLOCK_SIZE               = 16384 // 2^14
)

type PieceManager interface {
	GetBitField() []byte
	PeerStopped(id string)
	PieceHave(id string, pieceIndex int)
	BlockWritten(id string, pieceIndex, blockIndex int) error
	SendBlockRequests(id string, peer Peer, peerBitfield bitmap.Bitmap)
}

// How this module works
// Core - Maintains information of all peers (e.g. current outstanding pieces,
// peer bitfield, choke state) to ultimately figure out which pieces
// the client should choose to download based on piece rarity of swarm.

// Peer connections -
// -- PEER --
// choke state = if chocked clear outstanding field of peer state, if unchoked
// figure out most rare pieces and send requests (if outstanding requests isn't
// maxed)
// peer have message = send more piece requests if not maxed (based on new
// peer bitfield)
// -- DISK --
// disk piece written - figure out which peer successfully downloaded a piece,
// and send more piece requests (and the number of outstanding piece requests
// has decremented)
// -- PEER MANAGER --
// new Peer - create new peer entry
// dead Peer - remove peer entry

// The peer

// How this module works - 2
// Request -
// SavePiece -
// PeerExit -
