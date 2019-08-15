package torrent

type diskPeerChans struct {
	blockResponse chan *block
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

type block struct {
	pieceIndex      int
	blockByteOffset int
	blockData       []byte
}

type disk struct {
	stats *torrentStats
	// peerChans *diskPeerChans
}

type torrentStats struct {
	uploaded   int
	downloaded int
	left       int
}

func newDisk() *disk {
	return &disk{
		stats: &torrentStats{},
	}
}

func (d *disk) read() {

}
