package torrent

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"

	bitmap "github.com/boljen/go-bitmap"
)

const (
	CHOKE          = 0
	UNCHOKE        = 1
	INTERESTED     = 2
	NOT_INTERESTED = 3
	HAVE           = 4
	BITFIELD       = 5
	REQUEST        = 6
	BLOCK          = 7
	CANCEL         = 8
	PORT           = 9
)

var (
	CLIENT_BLOCK_REQUEST_LENGTH = 16384 // 2^14
	BLOCK_READ_REQUEST_DELAY    = 5
)

type pieceDownload struct {
	pieceIndex        int
	numBlocksInPiece  int
	numBlocksRecieved int
	blocksRecieved    []bool
	data              []byte // block size * num blocks
	sha1              []byte
}

type peer struct {
	id    string
	state struct {
		peerInterested   bool
		clientInterested bool
		peerChoking      bool
		clientChoking    bool
	}
	conn    net.Conn
	torrent *Torrent
	quit    chan int
	disk    *disk
	// pieceMgr              PieceMgr
	clientBitfield        bitmap.Bitmap
	downloads             []*pieceDownload
	readRequestCancelChan map[string]chan int
	// peerBitfield   bitmap.Bitmap
}

func newPeer(
	id string,
	conn net.Conn,
	disk *disk,
	torrent *Torrent,
	quit chan int,
	clientBitField bitmap.Bitmap,
	toChokeChans *peerChokeChans) *peer {

	peer := &peer{
		id:                    id,
		conn:                  conn,
		torrent:               torrent,
		quit:                  quit,
		disk:                  disk,
		clientBitfield:        clientBitField,
		downloads:             make([]*pieceDownload, 0),
		readRequestCancelChan: make(map[string]chan int),
	}
	go peer.start()
	return peer
}

func (p *peer) stop() {
	// TODO: remove peer from peer manager
	fmt.Printf("peer %s: is being stopped\n", p.id)
	close(p.quit)
}

func (p *peer) start() {
	if p.conn == nil {
		conn, err := net.Dial("tcp4", p.id)
		if err != nil {
			return
		}
		p.conn = conn
	}

	type handshake struct {
		Len      uint8
		Protocol [19]byte
		Reserved [8]uint8
		InfoHash [20]byte
		PeerID   [20]byte
	}

	// send handshake
	hreq := &handshake{}
	hreq.Len = 19
	copy(hreq.Protocol[:], "BitTorrent protocol")
	copy(hreq.InfoHash[:], p.torrent.infoHash)
	copy(hreq.PeerID[:], PEER_ID)
	err := binary.Write(p.conn, binary.BigEndian, hreq)
	if err != nil {
		p.stop()
		return
	}

	// recieve handshake
	hresp := &handshake{}
	err = binary.Read(p.conn, binary.BigEndian, hresp)
	if err != nil {
		p.stop()
		return
	}
	if hresp.Len != 19 ||
		!bytes.Equal(hresp.InfoHash[:], p.torrent.infoHash) ||
		!bytes.Equal(hresp.Protocol[:], []byte("BitTorrent protocol")) {
		p.stop()
		return
	}

	// send bitfield
	bitfield := p.clientBitfield.Data(false)
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, int32(1+len(bitfield)))
	binary.Write(b, binary.BigEndian, uint8(BITFIELD))
	binary.Write(b, binary.BigEndian, bitfield)
	p.sendMessage(b.Bytes())

	// handle all subsequent messages
	for {
		var length int64
		binary.Read(p.conn, binary.BigEndian, &length)
		var ID byte
		binary.Read(p.conn, binary.BigEndian, &ID)
		payload := make([]byte, length-1)
		binary.Read(p.conn, binary.BigEndian, &payload)
		p.decodeMessage(ID, bytes.NewBuffer(payload))
	}
}

func (p *peer) decodeMessage(messageID byte, payload *bytes.Buffer) {
	// TODO - maintain peer bitfield
	switch messageID {
	case CHOKE:
		if !p.state.peerChoking {
			p.state.peerChoking = true
			// go func() {
			// 	p.toChokeChans.clientChokeStateChan <- &chokeState{
			// 		peerID:   p.id,
			// 		isChoked: true,
			// 	}
			// }()
			// TODO - Choke Controller should eliminate pending requests
			// 	// clear unfinished work
		}
	case UNCHOKE:
		if p.state.peerChoking {
			p.state.peerChoking = false
			// go func() {
			// 	p.toChokeChans.clientChokeStateChan <- &chokeState{
			// 		peerID:   p.id,
			// 		isChoked: false,
			// 	}
			// }()
		}
	case INTERESTED:
		p.state.peerInterested = true
		// p.sendUnchoke()
		// p.connState.peerChoked = false
	case NOT_INTERESTED:
		p.state.peerInterested = false
		// p.sendChoke()
		// p.connState.peerChoked = true
	case HAVE:
		// var pieceIndex int
		// binary.Read(payload, binary.BigEndian, &pieceIndex)

		// go func() {
		// 	p.toChokeChans.peerHaveMessagesChan <- &peerHaveMessages{
		// 		peerID:       p.id,
		// 		pieceIndices: []int{pieceIndex}}
		// }()
	case BITFIELD:
		// peerHaveMessages := &peerHaveMessages{}
		// peerHaveMessages.peerID = p.id

		// peerBitfield := payload.Bytes()
		// for pieceIndex := 0; pieceIndex < p.torrent.numPieces; pieceIndex++ {
		// 	havePiece := bitmap.Get(peerBitfield, pieceIndex)
		// 	if havePiece {
		// 		peerHaveMessages.pieceIndices = append(peerHaveMessages.pieceIndices, pieceIndex)
		// 	}
		// }
		// go func() { p.toChokeChans.peerHaveMessagesChan <- peerHaveMessages }()
	case REQUEST:
		// brr := &blockReadRequest{}
		// binary.Read(payload, binary.BigEndian, &brr.pieceIndex)
		// binary.Read(payload, binary.BigEndian, &brr.blockByteOffset)
		// binary.Read(payload, binary.BigEndian, &brr.length)
		//brr.resp = p.fromDiskChans.blockResponse

		// requestID := strconv.Itoa(brr.pieceIndex) + strconv.Itoa(brr.blockByteOffset) + strconv.Itoa(brr.length)
		// quit := make(chan int)
		// go func() {
		// 	select {
		// 	case <-quit:
		// 		return
		// 	case <-time.After(BLOCK_READ_REQUEST_DELAY):
		// 		p.diskRequestCancellationChannel[requestID] = nil
		// 		p.toDiskChans.blockReadRequestChan <- brr
		// 	}
		// }()
		// p.diskRequestCancellationChannel[requestID] = quit

	case BLOCK:
		// b := &block{}
		// binary.Read(payload, binary.BigEndian, b.pieceIndex)
		// binary.Read(payload, binary.BigEndian, b.blockByteOffset)
		// binary.Read(payload, binary.BigEndian, b.blockData)

		// download, i := p.getPieceDownload(b.pieceIndex)
		// blockIndex := b.blockByteOffset / CLIENT_BLOCK_REQUEST_LENGTH

		// if download == nil {
		// 	log.Printf("Ignoring incoming block message for piece not being downloaded")
		// 	return
		// } else if b.blockByteOffset%CLIENT_BLOCK_REQUEST_LENGTH != 0 ||
		// 	blockIndex > 0 || blockIndex < 10 {
		// 	log.Printf("Illegal block byte offset within piece")
		// 	p.stop()
		// 	return
		// } else if download.blocksRecieved[blockIndex] {
		// 	log.Printf("Ignoring incoming block message for already downloaded block")
		// 	return
		// }

		// copy(download.data[b.blockByteOffset:], b.blockData)
		// download.numBlocksRecieved++
		// download.blocksRecieved[blockIndex] = true

		// // All blocks of piece have been downloaded
		// if download.numBlocksRecieved == download.numBlocksInPiece {
		// 	// Remove download from pending/outstanding downloads
		// 	p.removePieceDownloadByIndex(i)
		// 	// Check actual piece SHA1 against its expected value
		// 	pieceSHA1 := sha1.Sum(download.data)
		// 	if !bytes.Equal(pieceSHA1[:], download.sha1) {
		// 		log.Printf("peer %s: checksum of downloaded piece %d doesn't match\n", p.id, b.pieceIndex)
		// 		p.stop()
		// 		break
		// 	}
		// 	// Request to write piece to disk
		// 	p.toDiskChans.pieceWriteRequestChan <- &pieceWriteRequest{
		// 		pieceIndex: download.pieceIndex,
		// 		data:       download.data,
		// 	}
		// } else {
		// 	// TODO: requests sent that haven't been processed ?
		// 	//p.sendQueuedBlockRequests()
		// }
	case CANCEL:
		// brr := &blockReadRequest{}
		// binary.Read(payload, binary.BigEndian, &brr.pieceIndex)
		// binary.Read(payload, binary.BigEndian, &brr.blockByteOffset)
		// binary.Read(payload, binary.BigEndian, &brr.length)
		// requestID := strconv.Itoa(brr.pieceIndex) + strconv.Itoa(brr.blockByteOffset) + strconv.Itoa(brr.length)
		// quitC := p.diskRequestCancellationChannel[requestID]
		// if quitC != nil {
		// 	close(quitC)
		// }
	case PORT:
		// TODO: DHT (BEP 0005)
	}
}

func (p *peer) getPieceDownload(pieceIndex int) (*pieceDownload, int) {
	for i, pieceDownload := range p.downloads {
		if pieceIndex == pieceDownload.pieceIndex {
			return pieceDownload, i
		}
	}
	return nil, 0
}

func (p *peer) removePieceDownloadByIndex(i int) {
	p.downloads = append(p.downloads[:i], p.downloads[i+1:]...)
}

func (p *peer) sendBitfield() {

}

func (p *peer) sendChoke() {
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, uint8(CHOKE))
	p.sendMessage(b.Bytes())
}

func (p *peer) sendUnchoke() {
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, uint8(UNCHOKE))
	p.sendMessage(b.Bytes())
}

func (p *peer) sendMessage(msg []byte) {
	n, err := p.conn.Write(msg)
	if err != nil {
		p.stop()
	}
}

func (p *peer) updateBitField(havePieces []*havePiece) {
	for _, havePiece := range havePieces {
		p.clientBitfield.Set(havePiece.pieceIndex, true)
	}
}
