package peer

import (
	"net"
	"sync"

	"github.com/Charana123/torrent/go-torrent/stats"
	"github.com/Charana123/torrent/go-torrent/storage"

	"github.com/Charana123/torrent/go-torrent/piece"

	"github.com/Charana123/torrent/go-torrent/torrent"
)

type PeerManager interface {
	AddPeer(id string, conn net.Conn)
	RemovePeer(id string)
	GetPeerList() []*PeerInfo
	StopPeers()
}

type peerManager struct {
	sync.RWMutex
	torrent  *torrent.Torrent
	pieceMgr piece.PieceManager
	storage  storage.Storage
	stats    stats.Stats
	peers    map[string]Peer
	numPeers int
	maxPeers int
	quit     chan int
}

func NewPeerManager(
	torrent *torrent.Torrent,
	pieceMgr piece.PieceManager,
	storage storage.Storage,
	stats stats.Stats) PeerManager {

	return &peerManager{
		torrent:  torrent,
		pieceMgr: pieceMgr,
		storage:  storage,
		stats:    stats,
	}
}

func (pm *peerManager) StopPeers() {
	pm.RLock()
	defer pm.RUnlock()

	for _, peer := range pm.peers {
		peer.Stop()
	}
}

func (pm *peerManager) GetPeerList() []*PeerInfo {
	pm.RLock()
	defer pm.RUnlock()

	peers := []*PeerInfo{}
	for _, peer := range pm.peers {
		pi := &PeerInfo{}
		pi.id, pi.state, pi.wire, pi.lastPiece = peer.GetPeerInfo()
		pi.speed = 0 // TODO

		peers = append(peers, pi)
	}
	return peers
}

func (pm *peerManager) AddPeer(id string, conn net.Conn) {
	pm.Lock()
	defer pm.Unlock()

	if pm.numPeers > pm.maxPeers {
		// Connected to too many peers
		return
	}
	if _, ok := pm.peers[id]; ok {
		// Already connected to peer
		return
	}

	// peer = newPeer(
	// 	peer,
	// 	pm.torrent,
	// 	pm.quit,
	// 	pm.chokePeerChans,
	// )
	// notify choke chan ?
	// go func() { pm.chokeChans.newPeer <- fromChokeChans }()

	// pm.peers[id] = peer
	// pm.numPeers++
}

func (pm *peerManager) RemovePeer(id string) {
	pm.Lock()
	defer pm.Unlock()

	delete(pm.peers, id)
	pm.numPeers--
}
