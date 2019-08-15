package torrent

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	bitmap "github.com/boljen/go-bitmap"
)

const (
	KEEP_ALIVE     = -1
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
	BLOCK_READ_REQUEST_DELAY    = time.Second * time.Duration(5)
)

type chokeState struct {
	peerID   string
	isChoked bool
}

type peerHaveMessages struct {
	peerID       string
	pieceIndices []int
}

type peerChokeChans struct {
	clientChokeStateChan chan *chokeState
	peerHaveMessagesChan chan *peerHaveMessages
}

type peerDiskChans struct {
	blockReadRequestChan  chan *blockReadRequest
	pieceWriteRequestChan chan *pieceWriteRequest
}

type pieceDownload struct {
	pieceIndex        int
	numBlocksInPiece  int
	numBlocksRecieved int
	blocksRecieved    []bool
	data              []byte // block size * num blocks
	sha1              []byte
}

type peer struct {
	id        string
	conn      net.Conn
	connState struct {
		clientChoked     bool
		peerChoked       bool
		peerInterested   bool
		clientInterested bool
	}
	torrent                        *Torrent
	quit                           chan int
	toChokeChans                   *peerChokeChans
	fromChokeChans                 *chokePeerChans
	toDiskChans                    *peerDiskChans
	fromDiskChans                  *diskPeerChans
	clientBitfield                 bitmap.Bitmap
	downloads                      []*pieceDownload
	diskRequestCancellationChannel map[string]chan int
	// peerBitfield   bitmap.Bitmap
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

type handshake struct {
	Len      uint8
	Protocol [19]byte
	Reserved [8]uint8
	InfoHash [20]byte
	PeerID   [20]byte
}

func (p *peer) stop() {
	fmt.Printf("peer %s: is being stopped\n", p.id)
	close(p.quit)
}

func (p *peer) updateBitField(havePieces []*havePiece) {
	for _, havePiece := range havePieces {
		p.clientBitfield.Set(havePiece.pieceIndex, true)
	}
}

func (p *peer) sendBitfield() {
	bitmap := p.clientBitfield.Data(false)
	p.sendMessage(BITFIELD, bitmap)
}

func (p *peer) sendChoke() {
	p.sendMessage(CHOKE, nil)
}

func (p *peer) sendUnchoke() {
	p.sendMessage(UNCHOKE, nil)
}

func (p *peer) sendMessage(messageID int, data interface{}) {

}

func (p *peer) handshake() {
	handshake := &handshake{}
	handshake.Len = 19
	copy(handshake.Protocol[:], "BitTorrent protocol")
	copy(handshake.InfoHash[:], p.torrent.infoHash)
	copy(handshake.PeerID[:], PEER_ID)

	err := binary.Write(p.conn, binary.BigEndian, handshake)
	if err != nil {
		p.stop()
		return
	}
}

func (p *peer) decodeMessage(messageID byte, payload *bytes.Buffer) {
	// TODO - maintain peer bitfield
	switch messageID {
	case CHOKE:
		if !p.connState.clientChoked {
			p.connState.clientChoked = true
			go func() {
				p.toChokeChans.clientChokeStateChan <- &chokeState{
					peerID:   p.id,
					isChoked: true,
				}
			}()
			// TODO - Choke Controller should eliminate pending requests
			// 	// clear unfinished work
		}
	case UNCHOKE:
		if p.connState.clientChoked {
			p.connState.clientChoked = false
			go func() {
				p.toChokeChans.clientChokeStateChan <- &chokeState{
					peerID:   p.id,
					isChoked: false,
				}
			}()
		}
	case INTERESTED:
		p.connState.peerInterested = true
		p.sendUnchoke()
		p.connState.peerChoked = false
	case NOT_INTERESTED:
		p.connState.peerInterested = false
		p.sendChoke()
		p.connState.peerChoked = true
	case HAVE:
		var pieceIndex int
		binary.Read(payload, binary.BigEndian, &pieceIndex)

		go func() {
			p.toChokeChans.peerHaveMessagesChan <- &peerHaveMessages{
				peerID:       p.id,
				pieceIndices: []int{pieceIndex}}
		}()
	case BITFIELD:
		peerHaveMessages := &peerHaveMessages{}
		peerHaveMessages.peerID = p.id

		peerBitfield := payload.Bytes()
		for pieceIndex := 0; pieceIndex < p.torrent.numPieces; pieceIndex++ {
			havePiece := bitmap.Get(peerBitfield, pieceIndex)
			if havePiece {
				peerHaveMessages.pieceIndices = append(peerHaveMessages.pieceIndices, pieceIndex)
			}
		}
		go func() { p.toChokeChans.peerHaveMessagesChan <- peerHaveMessages }()
	case REQUEST:
		brr := &blockReadRequest{}
		binary.Read(payload, binary.BigEndian, &brr.pieceIndex)
		binary.Read(payload, binary.BigEndian, &brr.blockByteOffset)
		binary.Read(payload, binary.BigEndian, &brr.length)
		brr.resp = p.fromDiskChans.blockResponse

		requestID := strconv.Itoa(brr.pieceIndex) + strconv.Itoa(brr.blockByteOffset) + strconv.Itoa(brr.length)
		quit := make(chan int)
		go func() {
			select {
			case <-quit:
				return
			case <-time.After(BLOCK_READ_REQUEST_DELAY):
				p.diskRequestCancellationChannel[requestID] = nil
				p.toDiskChans.blockReadRequestChan <- brr
			}
		}()
		p.diskRequestCancellationChannel[requestID] = quit

	case BLOCK:
		b := &block{}
		binary.Read(payload, binary.BigEndian, b.pieceIndex)
		binary.Read(payload, binary.BigEndian, b.blockByteOffset)
		binary.Read(payload, binary.BigEndian, b.blockData)

		download, i := p.getPieceDownload(b.pieceIndex)
		blockIndex := b.blockByteOffset / CLIENT_BLOCK_REQUEST_LENGTH

		if download == nil {
			log.Printf("Ignoring incoming block message for piece not being downloaded")
			return
		} else if b.blockByteOffset%CLIENT_BLOCK_REQUEST_LENGTH != 0 ||
			blockIndex > 0 || blockIndex < 10 {
			log.Printf("Illegal block byte offset within piece")
			p.stop()
			return
		} else if download.blocksRecieved[blockIndex] {
			log.Printf("Ignoring incoming block message for already downloaded block")
			return
		}

		copy(download.data[b.blockByteOffset:], b.blockData)
		download.numBlocksRecieved++
		download.blocksRecieved[blockIndex] = true

		// All blocks of piece have been downloaded
		if download.numBlocksRecieved == download.numBlocksInPiece {
			// Remove download from pending/outstanding downloads
			p.removePieceDownloadByIndex(i)
			// Check actual piece SHA1 against its expected value
			pieceSHA1 := sha1.Sum(download.data)
			if !bytes.Equal(pieceSHA1[:], download.sha1) {
				log.Printf("peer %s: checksum of downloaded piece %d doesn't match\n", p.id, b.pieceIndex)
				p.stop()
				break
			}
			// Request to write piece to disk
			p.toDiskChans.pieceWriteRequestChan <- &pieceWriteRequest{
				pieceIndex: download.pieceIndex,
				data:       download.data,
			}
		} else {
			// TODO: requests sent that haven't been processed ?
			//p.sendQueuedBlockRequests()
		}
	case CANCEL:
		brr := &blockReadRequest{}
		binary.Read(payload, binary.BigEndian, &brr.pieceIndex)
		binary.Read(payload, binary.BigEndian, &brr.blockByteOffset)
		binary.Read(payload, binary.BigEndian, &brr.length)
		requestID := strconv.Itoa(brr.pieceIndex) + strconv.Itoa(brr.blockByteOffset) + strconv.Itoa(brr.length)
		quitC := p.diskRequestCancellationChannel[requestID]
		if quitC != nil {
			close(quitC)
		}
	case PORT:
		// TODO: DHT (BEP 0005)
	}
}

func (p *peer) handleIncomingMessages() {

	// handle handshake
	handshake := &handshake{}
	err := binary.Read(p.conn, binary.BigEndian, handshake)
	if err != nil {
		p.stop()
		return
	}
	if handshake.Len != 19 ||
		bytes.Equal(handshake.InfoHash[:], p.torrent.infoHash) ||
		bytes.Equal(handshake.Protocol[:], []byte("BitTorrent protocol")) {
		p.stop()
		return
	}

	func() {
		for {
			var length int64
			binary.Read(p.conn, binary.BigEndian, &length)
			var ID byte
			binary.Read(p.conn, binary.BigEndian, &ID)
			payload := make([]byte, length)
			binary.Read(p.conn, binary.BigEndian, &payload)
			p.decodeMessage(ID, bytes.NewBuffer(payload))
		}
	}()
}

func (p *peer) start() {

	p.handshake()
	p.updateBitField(<-p.fromChokeChans.havePiece)
	p.sendBitfield()
	p.handleIncomingMessages()

	// send handshake to peer
	// obtain bitfield from choke algorithm
	// send bitfield to peer
	// spawn thread to process incoming messages
	// use this thread to process other events
}
