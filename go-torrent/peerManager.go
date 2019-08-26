package torrent

import "sync"

type PeerManager interface {
	AddPeer(*peer)
	GetPeerList() []*PeerInfo
}

type peerManager struct {
	sync.Mutex

	// peer related
	peers    map[string]*peer
	numPeers int
	maxPeers int

	// channels
	trackerChans   *trackerPeerMChans
	chokeChans     *peerMChokeChans
	chokePeerChans *peerChokeChans

	// other
	torrent *Torrent
	quit    chan int
}

func newPeerManager(
	torrent *Torrent,
	trackerChans *trackerPeerMChans,
	chokeChans *peerMChokeChans,
	chokePeerChans *peerChokeChans) PeerManager {

	pm := &peerManager{
		torrent:        torrent,
		trackerChans:   trackerChans,
		chokeChans:     chokeChans,
		chokePeerChans: chokePeerChans,
	}
	return pm
}

func (pm *peerManager) GetPeerList() []*PeerInfo {
	peers := []*PeerInfo{}
	for _, peer := range pm.peers {
		pi := &PeerInfo{}
		pi.id = peer.id
		pi.state = peer.state
		pi.peer = peer
		pi.lastPiece = 0 // TODO
		pi.speed = 0     // TODO

		peers = append(peers, pi)
	}
	return peers
}

func (pm *peerManager) AddPeer(peer *peer) {
	pm.Lock()
	defer pm.Unlock()

	if pm.numPeers > pm.maxPeers {
		// Connected to too many peers
		return
	}
	if _, ok := pm.peers[peer.id]; ok {
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

	pm.peers[peer.id] = peer
	pm.numPeers++
}

func (pm *peerManager) DeletePeer(id string) {
	pm.Lock()
	defer pm.Unlock()
	delete(pm.peers, id)
	pm.numPeers--
}
