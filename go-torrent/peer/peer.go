package peer

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"time"

	"github.com/jackpal/bencode-go"

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
	Stop(err error, preFunc func(), restart bool) bool
	GetPeerInfo() (id string, state connState, lastPiece int64)
	GetWire() wire.Wire
	SendUnchoke()
	SendChoke()
	StartDownloading(tor *torrent.Torrent)
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
	mdMgr                 piece.MetadataManager
	wire                  wire.Wire
	stats                 stats.Stats
	readRequestCancelChan map[string]chan int
	peerBitfield          *bitmap.Bitmap
	lastPiece             int64
	lastMessageSent       time.Time
	blockRecieved         bool
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
	tor *torrent.Torrent,
	mdMgr piece.MetadataManager,
	storage storage.Storage,
	peerMgr PeerManager,
	pieceMgr piece.PieceManager,
	stats stats.Stats) *peer {

	peer := &peer{
		id:                    id,
		wire:                  wire,
		torrent:               tor,
		mdMgr:                 mdMgr,
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

func (p *peer) SendUnchoke() {
	p.state.clientChoking = false
	err := p.wire.SendUnchoke()
	if p.Stop(err, nil, false) {
		return
	}
}

func (p *peer) SendChoke() {
	p.state.clientChoking = true
	err := p.wire.SendChoke()
	if p.Stop(err, nil, false) {
		return
	}
}

func (p *peer) GetWire() wire.Wire {
	return p.wire
}

func (p *peer) BanPeerThisInterval() {
	// This peer hasn't unchoked us after we've showed interest
	// Stop waiting for this peer
	go func() {
		<-time.After(time.Second * 5)
		if p.state.peerChoking {
			p.peerMgr.BanPeerThisInterval(p.id)
			fmt.Println("banning peer")
			p.Stop(fmt.Errorf("Redundant peer"), func() {}, false)
		}
	}()
}

func (p *peer) Stop(err error, preFunc func(), restart bool) bool {
	if !p.closed && err != nil {
		if preFunc != nil {
			preFunc()
		}
		p.closed = true
		if p.wire != nil {
			p.wire.Close()
		}
		p.peerMgr.RemovePeer(p.id)
		p.pieceMgr.PeerStopped(p.id, p.peerBitfield)
		if restart {
			fmt.Println("restarting peer")
			p.peerMgr.AddPeer(p.id, nil)
		}
		return true
	}
	return false
}

func (p *peer) GetPeerInfo() (string, connState, int64) {
	return p.id, p.state, p.lastPiece
}

func (p *peer) StartDownloading(tor *torrent.Torrent) {
	p.torrent = tor
	p.checkInterestForPeer()
}

func (p *peer) checkInterestForPeer() {
	// If client doesn't have piece in peer bitfield, become interested
	clientBitField := p.pieceMgr.GetBitField()
	for pieceIndex := 0; pieceIndex < p.torrent.NumPieces; pieceIndex++ {
		if p.checkInterestForPiece(p.peerBitfield, clientBitField, pieceIndex) {
			break
		}
	}
	if !p.state.clientInterested {
		// We aren't interested in this peer, we shouldn't waste network
		// and compute resources on it waiting for it to obtain a piece
		// the client doesn't have
		// note - makes the HAVE method redundant
		p.peerMgr.BanPeerThisInterval(p.id)
		p.Stop(fmt.Errorf("Redundant peer"), func() {}, false)
	}
}

func (p *peer) checkInterestForPiece(peerBitfield *bitmap.Bitmap, clientBitField []byte, pieceIndex int) (clientInterested bool) {
	if p.peerBitfield.Get(pieceIndex) && !bitmap.Get(clientBitField, pieceIndex) {
		if !p.state.clientInterested {
			p.state.clientInterested = true
			err := p.wire.SendInterested()
			p.BanPeerThisInterval()
			if p.Stop(err, nil, false) {
				return
			}
			return true
		}
	}
	return false
}

func (p *peer) Start() {
	if p.wire == nil {
		conn, err := net.DialTimeout("tcp4", p.id, time.Duration(2*time.Second))
		if p.Stop(err, nil, false) {
			return
		}
		p.wire = newWire(conn.(*net.TCPConn), time.Duration(time.Minute*2))
	}

	// send handshake
	err := p.wire.SendHandshake(19, "BitTorrent protocol", p.torrent.InfoHash, torrent.PEER_ID)
	if p.Stop(err, nil, false) {
		return
	}

	// recieve handshake
	length, protocol, reservedBytes, infoHash, _, err := p.wire.ReadHandshake()
	if p.Stop(err, nil, false) {
		return
	}
	if !p.closed &&
		(length != 19 ||
			protocol != "BitTorrent protocol" ||
			!bytes.Equal(infoHash, p.torrent.InfoHash)) {
		p.Stop(fmt.Errorf("Malformed handshake"), nil, false)
		return
	}

	if p.torrent == nil && reservedBytes[5]&0x10 > 0 {
		p.wire.SendExtended()
	}

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
	if p.Stop(err3, nil, false) {
		return
	}

	// handle all subsequent messages
	for {
		length, messageID, payload, err := p.wire.ReadMessage()
		if p.Stop(err, nil, false) {
			fmt.Println("PEER: ", p.id, " Peer disconnecting")
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
	case wire.EXTENDED:
		fmt.Println("peer: ", p.id, ", extended")
		var extendedMessage uint8
		binary.Read(payload, binary.BigEndian, &extendedMessage)
		if extendedMessage != 0 {
			p.Stop(fmt.Errorf("Extended Message ID is non-zero i.e. not a handshake message"), func() {}, false)
		}
		extendedHandshakePayload := &wire.ExtendedHandshakePayload{}
		bencode.Unmarshal(payload, &extendedHandshakePayload)
		p.wire.SetExtendedMessageMap(extendedHandshakePayload.M)

		extendedMessageID, ok := extendedHandshakePayload.M["ut_metadata"]
		if ok && extendedMessageID != 0 && p.torrent == nil {
			p.mdMgr.Init(extendedHandshakePayload.MetadataSize)
			p.mdMgr.SendPieceRequest(p.id, p.wire)
		}
	case wire.UT_METADATA:
		fmt.Println("peer: ", p.id, ", metadata")
		pl := payload.Bytes()
		payload.Write(pl)

		mm := &wire.MetadataMessage{}
		err := bencode.Unmarshal(payload, mm)
		if err != nil {
			p.Stop(fmt.Errorf("Malformed metadata response"), func() {}, false)
		}

		bencode.Marshal(payload, mm)
		if mm.MessageType != 1 {
			p.Stop(fmt.Errorf("Metadata request rejected"), func() {}, false)
		}

		pieceData := pl[payload.Len():]
		if mm.Piece < p.mdMgr.GetNumMetaPieces()-1 && len(pieceData) != piece.METADATA_PIECE_SIZE ||
			mm.Piece == p.mdMgr.GetNumMetaPieces()-1 && len(pieceData) != mm.TotalSize-(p.mdMgr.GetNumMetaPieces()-1)*piece.METADATA_PIECE_SIZE ||
			mm.Piece > p.mdMgr.GetNumMetaPieces()-1 {
			p.Stop(fmt.Errorf("Malformed metadata response"), func() {}, false)
		}
		if !p.mdMgr.WritePiece(mm.Piece, pieceData) {
			p.mdMgr.SendPieceRequest(p.id, p.wire)
		}
	case wire.CHOKE:
		fmt.Println("peer: ", p.id, ", CHOKE")
		if !p.state.peerChoking {
			p.state.peerChoking = true
			p.pieceMgr.PeerChoked(p.id)
		}
		if p.state.clientInterested && p.blockRecieved {
			// To maintain active connections i.e.
			// 1) interested and unchoked - we are downloading from peer
			// 2) uninterested and choked - we maintain an open connection waiting for a HAVE request to
			// become interested again
			// we must reconnect if a peer chokes us while we are interested and has
			// will most likely fullfil a block request (as it has done so in the past)
			p.Stop(fmt.Errorf("Restarting Peer"), func() {}, true)
		}
	case wire.UNCHOKE:
		fmt.Println("UNCHOKE")
		if p.state.peerChoking {
			p.state.peerChoking = false
			if p.state.clientInterested && !p.state.peerChoking {
				p.pieceMgr.SendBlockRequests(p.id, p.wire, p.peerBitfield)
			}
		}
	case wire.INTERESTED:
		fmt.Println("PEER_INTERESTED")
		p.state.peerInterested = true
	case wire.NOT_INTERESTED:
		p.state.peerInterested = false
	case wire.HAVE:
		var pieceIndex int
		binary.Read(payload, binary.BigEndian, &pieceIndex)
		p.pieceMgr.PieceHave(p.id, pieceIndex)
		p.peerBitfield.Set(pieceIndex, true)

		// If client doesn't have piece, become interested
		if p.torrent != nil {
			clientBitField := p.pieceMgr.GetBitField()
			p.checkInterestForPiece(p.peerBitfield, clientBitField, pieceIndex)
		}

	case wire.BITFIELD:
		fmt.Println("BITFIELD")
		peerBitfield := payload.Bytes()
		bitfield := bitmap.New(p.torrent.NumPieces)
		p.peerBitfield = &bitfield

		for pieceIndex := 0; pieceIndex < p.torrent.NumPieces; pieceIndex++ {
			havePiece := bitmap.Get(peerBitfield, (pieceIndex/8)*8+7-pieceIndex%8)
			if havePiece {
				p.peerBitfield.Set(pieceIndex, true)
				p.pieceMgr.PieceHave(p.id, pieceIndex)
			}
		}
		fmt.Println("peer: ", p.id, " bitfield: ", p.peerBitfield)

		if p.torrent != nil {
			p.checkInterestForPeer()
		}
	case wire.REQUEST:
		fmt.Print("REQUEST")
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
					if p.Stop(err, nil, false) {
						return
					}
					err = p.wire.SendBlock(pieceIndex, blockByteOffset, block)
					if p.Stop(err, nil, false) {
						return
					}
					p.stats.UpdatePeer(p.id, 0, length)
				}
			}()
			p.readRequestCancelChan[requestID] = quit
		} else {
			if p.Stop(fmt.Errorf("peer sent cancel when client was choking or peer wasn't interested"), nil, false) {
				return
			}
		}
	case wire.BLOCK:
		p.blockRecieved = true
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
			fmt.Println("PEER:", p.id, "PIECE", pieceIndex, "BLOCK", blockIndex)
			go func() {
				downloadedPiece, peers, err := p.pieceMgr.WriteBlock(p.id, pieceIndex, blockIndex, blockData)
				if p.Stop(err, func() {
					if downloadedPiece && peers != nil {
						p.peerMgr.BanPeers(peers)
					}
				}, false) {
					return
				}
				if downloadedPiece {
					p.peerMgr.BroadcastHave(pieceIndex)
				}
				p.stats.UpdatePeer(p.id, blockLength, 0)
				p.pieceMgr.SendBlockRequests(p.id, p.wire, p.peerBitfield)
			}()
			p.lastPiece = time.Now().Unix()
		}
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
			if p.Stop(fmt.Errorf("peer sent cancel when client was choking or peer wasn't interested"), nil, false) {
				return
			}
		}
	case wire.PORT:
		// TODO: DHT (BEP 0005)
	}
}
