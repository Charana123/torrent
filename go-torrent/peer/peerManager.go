package peer

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/Charana123/torrent/go-torrent/stats"
	"github.com/Charana123/torrent/go-torrent/storage"
	"github.com/Charana123/torrent/go-torrent/wire"

	"github.com/Charana123/torrent/go-torrent/piece"

	"github.com/Charana123/torrent/go-torrent/torrent"
	mapset "github.com/deckarep/golang-set"
)

const (
	PEER_TIMEOUT = 120
)

type PeerManager interface {
	AddPeer(id string, conn net.Conn)
	RemovePeer(id string)
	GetPeerList() []Peer
	StopPeers()
	BroadcastHave(pieceIndex int)
	BanPeers(peers mapset.Set)
}

type peerManager struct {
	sync.RWMutex
	torrent     *torrent.Torrent
	pieceMgr    piece.PieceManager
	storage     storage.Storage
	stats       stats.Stats
	peers       map[string]Peer
	numPeers    int
	maxPeers    int
	bannedPeers mapset.Set
}

func NewPeerManager(
	torrent *torrent.Torrent,
	pieceMgr piece.PieceManager,
	storage storage.Storage,
	stats stats.Stats) PeerManager {

	return &peerManager{
		torrent:     torrent,
		pieceMgr:    pieceMgr,
		storage:     storage,
		stats:       stats,
		peers:       make(map[string]Peer),
		bannedPeers: mapset.NewSet(),
		maxPeers:    100,
	}
}

func (pm *peerManager) BanPeers(peers mapset.Set) {
	pm.Lock()
	defer pm.Unlock()

	pm.bannedPeers.Union(peers)
}

func (pm *peerManager) BroadcastHave(pieceIndex int) {
	pm.RLock()
	defer pm.RUnlock()

	for _, peer := range pm.peers {
		wire := peer.GetWire()
		if wire != nil {
			wire.SendHave(pieceIndex)
		}
	}
}

func (pm *peerManager) StopPeers() {
	pm.RLock()
	defer pm.RUnlock()

	for _, peer := range pm.peers {
		peer.Stop(fmt.Errorf("Peer gracefully shutdown"), nil)
	}
}

func (pm *peerManager) GetPeerList() []Peer {
	pm.RLock()
	defer pm.RUnlock()

	peers := []Peer{}
	for _, peer := range pm.peers {
		peers = append(peers, peer)
	}
	return peers
}

func (pm *peerManager) AddPeer(id string, conn net.Conn) {
	pm.Lock()
	defer pm.Unlock()

	if pm.bannedPeers.Contains(id) {
		// Peer has been banned
		return
	}
	if pm.numPeers > pm.maxPeers {
		// Connected to too many peers
		return
	}
	if _, ok := pm.peers[id]; ok {
		// Already connected to peer
		return
	}

	w := (wire.Wire)(nil)
	if conn != nil {
		w = wire.NewWire(conn.(*net.TCPConn), time.Duration(time.Second*PEER_TIMEOUT))
	}
	peer := NewPeer(
		id,
		w,
		pm.torrent,
		pm.storage,
		pm,
		pm.pieceMgr,
		pm.stats,
	)
	pm.peers[id] = peer
	pm.numPeers++
	go peer.Start()
}

func (pm *peerManager) RemovePeer(id string) {
	pm.Lock()
	defer pm.Unlock()

	delete(pm.peers, id)
	pm.numPeers--
}
