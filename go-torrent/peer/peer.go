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

type Peer interface {
	Stop()
}

var newWire = wire.NewWire

func (p *peer) Stop() {
	go func() {
		p.peerMgr.RemovePeer(p.id)
		p.pieceMgr.PeerStopped(p.id, p.peerBitfield)
	}()
	p.closed = true
	p.wire.Close()
}

type peer struct {
	id                    string
	state                 connState
	closed                bool
	disk                  disk.Disk
	torrent               *torrent.Torrent
	wire                  wire.Wire
	peerMgr               PeerManager
	pieceMgr              piece.PieceManager
	readRequestCancelChan map[string]chan int
	peerBitfield          *bitmap.Bitmap
}

type connState struct {
	peerInterested   bool
	clientInterested bool
	peerChoking      bool
	clientChoking    bool
}

func NewPeer(
	id string,
	wire wire.Wire,
	torrent *torrent.Torrent,
	disk disk.Disk,
	peerMgr PeerManager,
	pieceMgr piece.PieceManager) *peer {

	peer := &peer{
		id:                    id,
		wire:                  wire,
		torrent:               torrent,
		disk:                  disk,
		peerMgr:               peerMgr,
		pieceMgr:              pieceMgr,
		readRequestCancelChan: make(map[string]chan int),
	}
	go peer.start()
	return peer
}

func (p *peer) start() {
	if p.wire == nil {
		conn, err := net.Dial("tcp4", p.id)
		if err != nil {
			p.Stop()
			return
		}
		p.wire = newWire(conn)
	}

	// send handshake
	err1 := p.wire.SendHandshake(19, "BitTorrent protocol", p.torrent.InfoHash, torrent.PEER_ID)
	if !p.closed && err1 != nil {
		p.Stop()
		fmt.Println("one")
		return
	}

	// recieve handshake
	length, protocol, infoHash, _, err2 := p.wire.ReadHandshake()
	if length != 19 ||
		protocol != "BitTorrent protocol" ||
		!bytes.Equal(infoHash, p.torrent.InfoHash) {
		fmt.Println("one1")
		p.Stop()
		return
	}
	if !p.closed && err2 != nil {
		fmt.Println("one2")
		p.Stop()
		return
	}

	// send bitfield
	bitfield := p.pieceMgr.GetBitField()
	err3 := p.wire.SendBitField(bitfield)
	if !p.closed && err3 != nil {
		p.Stop()
		fmt.Println("one3")
		return
	}

	// handle all subsequent messages
	for {
		length, messageID, payload, err := p.wire.ReadMessage()
		if !p.closed && err != nil {
			fmt.Println("one4")
			p.Stop()
			return
		}
		if length == 0 {
			// keep-alive message
			continue
		}
		p.decodeMessage(messageID, bytes.NewBuffer(payload))
	}
}

func (p *peer) decodeMessage(messageID uint8, payload *bytes.Buffer) {
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
		bitfield := bitmap.New(p.torrent.NumPieces)
		p.peerBitfield = &bitfield
		for pieceIndex := 0; pieceIndex < p.torrent.NumPieces; pieceIndex++ {
			havePiece := bitmap.Get(peerBitfield, pieceIndex)
			if havePiece {
				p.peerBitfield.Set(pieceIndex, true)
				p.pieceMgr.PieceHave(p.id, pieceIndex)
			}
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
