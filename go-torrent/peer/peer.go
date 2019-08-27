package peer

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/Charana123/torrent/go-torrent/disk"
	"github.com/Charana123/torrent/go-torrent/piece"
	"github.com/Charana123/torrent/go-torrent/torrent"
	"github.com/Charana123/torrent/go-torrent/wire"
	bitmap "github.com/boljen/go-bitmap"
)

var (
	BLOCK_READ_REQUEST_DELAY = 5
)

var dial = net.Dial

type Peer interface {
	Stop()
}

func (p *peer) Stop() {
	go func() {
		p.peerMgr.RemovePeer(p.id)
		p.pieceMgr.PeerStopped(p.id, p.peerBitfield)
	}()
	fmt.Printf("peer %s: is being stopped\n", p.id)
	close(p.quit)
}

type peer struct {
	id                    string
	state                 connState
	conn                  net.Conn
	disk                  disk.Disk
	torrent               *torrent.Torrent
	quit                  chan int
	wire                  wire.Wire
	peerMgr               PeerManager
	pieceMgr              piece.PieceManager
	readRequestCancelChan map[string]chan int
	peerBitfield          bitmap.Bitmap
}

type connState struct {
	peerInterested   bool
	clientInterested bool
	peerChoking      bool
	clientChoking    bool
}

func newPeer(
	id string,
	conn net.Conn,
	disk disk.Disk,
	torrent *torrent.Torrent,
	quit chan int,
	peerMgr PeerManager,
	pieceMgr piece.PieceManager) *peer {

	peer := &peer{
		id:                    id,
		conn:                  conn,
		disk:                  disk,
		torrent:               torrent,
		quit:                  quit,
		peerMgr:               peerMgr,
		pieceMgr:              pieceMgr,
		readRequestCancelChan: make(map[string]chan int),
	}
	go peer.start()
	return peer
}

func (p *peer) start() {
	if p.conn == nil {
		conn, err := dial("tcp4", p.id)
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
	copy(hreq.InfoHash[:], p.torrent.InfoHash)
	copy(hreq.PeerID[:], torrent.PEER_ID)
	err := binary.Write(p.conn, binary.BigEndian, hreq)
	if err != nil {
		p.Stop()
		return
	}

	// recieve handshake
	hresp := &handshake{}
	err = binary.Read(p.conn, binary.BigEndian, hresp)
	if err != nil {
		p.Stop()
		return
	}
	if hresp.Len != 19 ||
		!bytes.Equal(hresp.InfoHash[:], p.torrent.InfoHash) ||
		!bytes.Equal(hresp.Protocol[:], []byte("BitTorrent protocol")) {
		p.Stop()
		return
	}

	// send bitfield
	bitfield := p.pieceMgr.GetBitField()
	p.wire.SendBitField(bitfield)

	// handle all subsequent messages
	for {
		var length int
		binary.Read(p.conn, binary.BigEndian, &length)
		if length == 0 {
			// keep-alive message
			continue
		}
		var ID byte
		binary.Read(p.conn, binary.BigEndian, &ID)
		payload := make([]byte, length-1)
		binary.Read(p.conn, binary.BigEndian, &payload)
		p.decodeMessage(ID, bytes.NewBuffer(payload))
	}
}

func (p *peer) decodeMessage(messageID byte, payload *bytes.Buffer) {
	switch messageID {
	case wire.CHOKE:
		if !p.state.peerChoking {
			p.state.peerChoking = true
			go func() {
				p.pieceMgr.PeerChoked(p.id)
			}()
		}
	case wire.UNCHOKE:
		if p.state.peerChoking {
			p.state.peerChoking = false
			go func() {
				p.pieceMgr.SendBlockRequests(p.id, p.wire, p.peerBitfield)
			}()
		}
	case wire.INTERESTED:
		p.state.peerInterested = true
	case wire.NOT_INTERESTED:
		p.state.peerInterested = false
	case wire.HAVE:
		var pieceIndex int
		binary.Read(payload, binary.BigEndian, &pieceIndex)
		go func() {
			p.pieceMgr.PieceHave(p.id, pieceIndex)
		}()
		p.peerBitfield.Set(pieceIndex, true)

		// If client doesn't have piece, become interested
		if bitmap.Get(p.pieceMgr.GetBitField(), pieceIndex) {
			if !p.state.clientInterested {
				p.state.clientInterested = true
				p.wire.SendInterested()
			}
		}
	case wire.BITFIELD:
		peerBitfield := payload.Bytes()
		p.peerBitfield = bitmap.New(p.torrent.NumPieces)
		for pieceIndex := 0; pieceIndex < p.torrent.NumPieces; pieceIndex++ {
			havePiece := bitmap.Get(peerBitfield, pieceIndex)
			p.peerBitfield.Set(pieceIndex, havePiece)
		}

		// If client doesn't have piece in peer bitfield, become interested
		clientBitField := p.pieceMgr.GetBitField()
		for pieceIndex := 0; pieceIndex < p.peerBitfield.Len(); pieceIndex++ {
			if p.peerBitfield.Get(pieceIndex) {
				if !bitmap.Get(clientBitField, pieceIndex) {
					p.state.clientInterested = true
					break
				}
			}
		}
	case wire.REQUEST:
		if !p.state.clientChoking && p.state.peerInterested {
			var pieceIndex int
			binary.Read(payload, binary.BigEndian, &pieceIndex)
			var blockByteOffset int
			binary.Read(payload, binary.BigEndian, &blockByteOffset)
			var length int
			binary.Read(payload, binary.BigEndian, &length)

			requestID := strconv.Itoa(pieceIndex) + strconv.Itoa(blockByteOffset) + strconv.Itoa(length)
			quit := make(chan int)
			go func() {
				select {
				case <-quit:
					return
				case <-time.After(time.Duration(BLOCK_READ_REQUEST_DELAY) * time.Second):
					delete(p.readRequestCancelChan, requestID)
					block, err := p.disk.BlockReadRequest(pieceIndex, blockByteOffset, length)
					if err != nil {
						fmt.Println("invalid block request from peer")
						p.Stop()
					}
					p.wire.SendBlock(pieceIndex, blockByteOffset, block)
				}
			}()
			p.readRequestCancelChan[requestID] = quit
		} else {
			log.Println("peer sent request when client was choking or peer wasn't interested")
			p.Stop()
			return
		}
	case wire.BLOCK:
		var pieceIndex int
		binary.Read(payload, binary.BigEndian, pieceIndex)
		var blockByteOffset int
		binary.Read(payload, binary.BigEndian, blockByteOffset)
		var blockData []byte
		binary.Read(payload, binary.BigEndian, blockData)

		blockIndex := blockByteOffset / piece.BLOCK_SIZE
		go func() {
			err := p.pieceMgr.WriteBlock(p.id, pieceIndex, blockIndex, blockData)
			if err != nil {
				log.Println(err)
				p.Stop()
			}
			p.pieceMgr.SendBlockRequests(p.id, p.wire, p.peerBitfield)
		}()
	case wire.CANCEL:
		if !p.state.clientChoking && p.state.peerInterested {
			var pieceIndex int
			binary.Read(payload, binary.BigEndian, &pieceIndex)
			var blockByteOffset int
			binary.Read(payload, binary.BigEndian, &blockByteOffset)
			var length int
			binary.Read(payload, binary.BigEndian, &length)

			requestID := strconv.Itoa(pieceIndex) + strconv.Itoa(blockByteOffset) + strconv.Itoa(length)
			if quitC, ok := p.readRequestCancelChan[requestID]; ok {
				close(quitC)
			}
		} else {
			log.Println("peer sent cancel when client was choking or peer wasn't interested")
			p.Stop()
			return
		}
	case wire.PORT:
		// TODO: DHT (BEP 0005)
	}
}
