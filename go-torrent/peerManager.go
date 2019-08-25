package torrent

import "sync"

type PeerManager interface {
	AddPeer(*peer)
}

type PM struct {
	torrent        *Torrent
	trackerChans   *trackerPeerMChans
	chokeChans     *peerMChokeChans
	chokePeerChans *peerChokeChans

	peers    map[string]*peer
	numPeers int
	maxPeers int
	peerLock *sync.Mutex

	quit chan int
}

func newPeerManager(
	torrent *Torrent,
	trackerChans *trackerPeerMChans,
	chokeChans *peerMChokeChans,
	chokePeerChans *peerChokeChans) PeerManager {

	pm := &PM{
		torrent:        torrent,
		trackerChans:   trackerChans,
		chokeChans:     chokeChans,
		chokePeerChans: chokePeerChans,
		peerLock:       &sync.Mutex{},
	}
	return pm
}

func (pm *PM) AddPeer(peer *peer) {
	pm.peerLock.Lock()
	defer pm.peerLock.Unlock()

	if pm.numPeers > pm.maxPeers {
		// Connected to too many peers
		return
	}
	if _, ok := pm.peers[peer.id]; ok {
		// Already connected to peer
		return
	}

	peer = newPeer(
		peer,
		pm.torrent,
		pm.quit,
		pm.chokePeerChans,
	)
	// notify choke chan ?
	// go func() { pm.chokeChans.newPeer <- fromChokeChans }()

	pm.peers[peer.id] = peer
	pm.numPeers++
	go peer.start()
}
