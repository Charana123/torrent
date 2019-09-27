package peer

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/Charana123/torrent/go-torrent/piece"
	"github.com/Charana123/torrent/go-torrent/stats"
	"github.com/Charana123/torrent/go-torrent/storage"
	"github.com/Charana123/torrent/go-torrent/torrent"
	"github.com/Charana123/torrent/go-torrent/wire"
	bitmap "github.com/boljen/go-bitmap"
)

var (
	BLOCK_READ_REQUEST_DELAY = 5
)

type Peer interface {
	Start()
	Stop()
	GetPeerInfo() (id string, state connState, wire wire.Wire, lastPiece int64)
	GetWire() wire.Wire
}

var newWire = wire.NewWire

type peer struct {
	id                    string
	state                 connState
	closed                bool
	storage               storage.Storage
	torrent               *torrent.Torrent
	peerMgr               PeerManager
	pieceMgr              piece.PieceManager
	wire                  wire.Wire
	stats                 stats.Stats
	readRequestCancelChan map[string]chan int
	peerBitfield          *bitmap.Bitmap
	lastPiece             int64
	lastMessageSent       time.Time
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
	storage storage.Storage,
	peerMgr PeerManager,
	pieceMgr piece.PieceManager,
	stats stats.Stats) *peer {

	peer := &peer{
		id:                    id,
		wire:                  wire,
		torrent:               torrent,
		storage:               storage,
		peerMgr:               peerMgr,
		pieceMgr:              pieceMgr,
		stats:                 stats,
		readRequestCancelChan: make(map[string]chan int),
		state: connState{
			peerChoking:      true,
			clientChoking:    true,
			peerInterested:   false,
			clientInterested: false,
		},
	}
	return peer
}

func (p *peer) GetWire() wire.Wire {
	return p.wire
}

func (p *peer) Stop() {
	go func() {
		p.peerMgr.RemovePeer(p.id)
		p.pieceMgr.PeerStopped(p.id, p.peerBitfield)
	}()
	p.closed = true
	if p.wire != nil {
		fmt.Println("client stopped")
		p.wire.Close()
	}
}

func (p *peer) GetPeerInfo() (string, connState, wire.Wire, int64) {
	return p.id, p.state, p.wire, p.lastPiece
}

func (p *peer) Start() {
	if p.wire == nil {
		conn, err := net.DialTimeout("tcp4", p.id, time.Duration(2*time.Second))
		if !p.closed && err != nil {
			p.Stop()
			return
		}
		p.wire = newWire(conn.(*net.TCPConn), time.Duration(time.Second*2))
	}
	fmt.Println(p.id, "connected")

	// send handshake
	err := p.wire.SendHandshake(19, "BitTorrent protocol", p.torrent.InfoHash, torrent.PEER_ID)
	if !p.closed && err != nil {
		p.Stop()
		return
	}
	fmt.Println(p.id, "sent handshake")

	// recieve handshake
	length, protocol, infoHash, _, err := p.wire.ReadHandshake()
	if !p.closed &&
		(err != nil ||
			length != 19 ||
			protocol != "BitTorrent protocol" ||
			!bytes.Equal(infoHash, p.torrent.InfoHash)) {
		fmt.Println(err)
		p.Stop()
		return
	}
	fmt.Println(p.id, "recieved handshake")

	// keep-alive thread
	go func() {
		interval := time.Duration(time.Minute)
		for {
			now := <-time.After(interval)
			// Send a keep alive if we haven't sent a message in over a minute
			if p.wire.GetLastMessageSent().Before(now.Add(-interval)) {
				err := p.wire.SendKeepAlive()
				if err != nil {
					return
				}
			}
		}
	}()

	// send bitfield
	bitfield := p.pieceMgr.GetBitField()
	err3 := p.wire.SendBitField(bitfield)
	if !p.closed && err3 != nil {
		p.Stop()
		return
	}
	fmt.Println(p.id, "sent bitfield")

	// handle all subsequent messages
	for {
		length, messageID, payload, err := p.wire.ReadMessage()
		if !p.closed && err != nil {
			fmt.Println("error", err)
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
		fmt.Println(p.id, "CHOKE")
		if !p.state.peerChoking {
			p.state.peerChoking = true
			go func() {
				p.pieceMgr.PeerChoked(p.id)
			}()
		}
	case wire.UNCHOKE:
		fmt.Println(p.id, "UNCHOKE")
		if p.state.peerChoking {
			p.state.peerChoking = false
			go func() {
				p.pieceMgr.SendBlockRequests(p.id, p.wire, p.peerBitfield)
			}()
		}
	case wire.INTERESTED:
		fmt.Println(p.id, "INTERESTED")
		p.state.peerInterested = true
	case wire.NOT_INTERESTED:
		fmt.Println(p.id, "NOT_INTERESTED")
		p.state.peerInterested = false
	case wire.HAVE:
		fmt.Println(p.id, "HAVE")
		var pieceIndex int
		binary.Read(payload, binary.BigEndian, &pieceIndex)
		go func() {
			p.pieceMgr.PieceHave(p.id, pieceIndex)
		}()
		p.peerBitfield.Set(pieceIndex, true)

		// If client doesn't have piece, become interested
		if !bitmap.Get(p.pieceMgr.GetBitField(), pieceIndex) {
			if !p.state.clientInterested {
				p.state.clientInterested = true
				err := p.wire.SendInterested()
				fmt.Println(p.id, "client interested")
				if !p.closed && err != nil {
					fmt.Println("error")
					p.Stop()
					return
				}
			}
		}
	case wire.BITFIELD:
		fmt.Println(p.id, "BITFIELD")
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
		for pieceIndex := 0; pieceIndex < p.torrent.NumPieces; pieceIndex++ {
			if p.peerBitfield.Get(pieceIndex) {
				if !bitmap.Get(clientBitField, pieceIndex) {
					p.state.clientInterested = true
					err := p.wire.SendInterested()
					fmt.Println(p.id, "client interested")
					if !p.closed && err != nil {
						fmt.Println("error")
						p.Stop()
						return
					}
					break
				}
			}
		}
	case wire.REQUEST:
		fmt.Println(p.id, "REQUEST")
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
					block, err := p.storage.BlockReadRequest(pieceIndex, blockByteOffset, length)
					if !p.closed && err != nil {
						p.Stop()
						return
					}
					err = p.wire.SendBlock(pieceIndex, blockByteOffset, block)
					if !p.closed && err != nil {
						p.Stop()
						return
					}
					p.stats.UpdatePeer(p.id, 0, length)
				}
			}()
			p.readRequestCancelChan[requestID] = quit
		} else {
			log.Println("peer sent request when client was choking or peer wasn't interested")
			if !p.closed {
				p.Stop()
				return
			}
		}
	case wire.BLOCK:
		if !p.state.peerChoking && p.state.clientInterested {
			var pi int32
			binary.Read(payload, binary.BigEndian, &pi)
			pieceIndex := int(pi)
			var bbo int32
			binary.Read(payload, binary.BigEndian, &bbo)
			blockByteOffset := int(bbo)
			blockData, _ := ioutil.ReadAll(payload)
			blockLength := len(blockData)

			blockIndex := blockByteOffset / piece.BLOCK_SIZE
			fmt.Println(p.id, "BLOCK", pieceIndex, blockIndex)
			go func() {
				downloadedPiece, piece, peers, err := p.pieceMgr.WriteBlock(p.id, pieceIndex, blockIndex, blockData)
				if !p.closed && err != nil {
					p.Stop()
					return
				}
				if downloadedPiece {
					expectedChecksum := []byte(p.torrent.MetaInfo.Info.Pieces[20*pieceIndex : 20*(pieceIndex+1)])
					actualChecksum := sha1.Sum(piece)
					if !bytes.Equal(expectedChecksum[:], actualChecksum[:]) {
						p.peerMgr.BanPeers(peers)
						if !p.closed {
							p.Stop()
							return
						}
					}
					err = p.storage.WritePieceRequest(pieceIndex, piece)
					if !p.closed && err != nil {
						p.Stop()
						return
					}
					p.peerMgr.BroadcastHave(pieceIndex)
				}
				p.stats.UpdatePeer(p.id, blockLength, 0)
				p.pieceMgr.SendBlockRequests(p.id, p.wire, p.peerBitfield)
			}()
			p.lastPiece = time.Now().Unix()
		}
	case wire.CANCEL:
		fmt.Println(p.id, "CANCEL")
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
			if !p.closed {
				p.Stop()
				return
			}
		}
	case wire.PORT:
		fmt.Println(p.id, "PORT")
		// TODO: DHT (BEP 0005)
	}
}
