package torrent

// Peer -> Disk
type peerDiskChans struct {
	blockReadRequestChan  chan *blockReadRequest
	pieceWriteRequestChan chan *pieceWriteRequest
}

type blockReadRequest struct {
	pieceIndex      int
	blockByteOffset int
	length          int
	resp            chan *block
}

type pieceWriteRequest struct {
	pieceIndex int
	data       []byte
	// response channel ?
}

// Disk -> Peer
type diskPeerChans struct {
	blockResponse chan *block
}

type block struct {
	pieceIndex      int
	blockByteOffset int
	blockData       []byte
}

// Server -> Peer Manager
type serverPeerMChans struct {
	peers chan *peer
}

type trackerStats struct {
	leechers int32
	seeders  int32
}

// Tracker -> Peer Manager
type trackerPeerMChans struct {
	// peer data (remote ip and port) from tracker
	peers chan *peer
}

//

type progressStats struct {
	uploaded   int
	downloaded int
	left       int
}

// ======== CHOKE ALGORITHM ======================

// Peer -> Choke Algoritm
type peerChokeChans struct {
	// To notify choke algorithm when peer choke state changed
	clientChokeStateChan chan *chokeState
	// To notify choke algorithm when peer sends have or bitfield message
	peerHaveMessagesChan chan *peerHaveMessages
}

type chokeState struct {
	peerID   string
	isChoked bool
}

type peerHaveMessages struct {
	peerID       string
	pieceIndices []int
}

// Peer Manager -> Choke Algorithm
type peerMChokeChans struct {
	newPeer chan *chokePeerChans
}

type chokePeerChans struct {
	havePiece chan []*havePiece
}

type havePiece struct {
	pieceIndex int
}
