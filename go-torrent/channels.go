package torrent

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

// ======== DISK ======================

type blockReadRequest struct {
	pieceIndex      int
	blockByteOffset int
	length          int
}

type blockReadResponse struct {
	pieceIndex      int
	blockByteOffset int
	blockData       []byte
}

type pieceWriteRequest struct {
	pieceIndex int
	data       []byte
}
