package torrent

import (
	"net"
)

type peerManager struct {
	torrent        *Torrent
	serverChans    *serverPeerMChans
	trackerChans   *trackerPeerMChans
	chokeChans     *peerMChokeChans
	chokePeerChans *peerChokeChans
	diskPeerChans  *peerDiskChans
	peers          map[string]*peer
	numPeers       int
	maxPeers       int
	quit           chan int
}

func newPeerManager(
	torrent *Torrent,
	serverChans *serverPeerMChans,
	trackerChans *trackerPeerMChans,
	chokeChans *peerMChokeChans,
	chokePeerChans *peerChokeChans,
	diskPeerChans *peerDiskChans) *peerManager {

	pm := &peerManager{
		torrent:        torrent,
		serverChans:    serverChans,
		trackerChans:   trackerChans,
		chokeChans:     chokeChans,
		chokePeerChans: chokePeerChans,
		diskPeerChans:  diskPeerChans,
	}
	return pm
}

func (pm *peerManager) start() {
	for {
		select {
		case peer := <-pm.serverChans.peers:
			if pm.numPeers > pm.maxPeers {
				// Connected to too many peers
				break
			}
			if _, ok := pm.peers[peer.id]; ok {
				// Already connected to peer
				break
			}

			peer = newPeer(
				peer,
				pm.torrent,
				pm.quit,
				pm.chokePeerChans,
				pm.diskPeerChans,
			)
			// notify choke chan ?
			// go func() { pm.chokeChans.newPeer <- fromChokeChans }()

			pm.peers[peer.id] = peer
			pm.numPeers++
			go peer.start()
		case peer := <-pm.trackerChans.peers:
			if pm.numPeers > pm.maxPeers {
				// Connected to too many peers
				break
			}
			if _, ok := pm.peers[peer.id]; ok {
				// Already connected to peer
				break
			}

			go func() {
				conn, err := net.Dial("tcp4", peer.id)
				if err != nil {
					return
				}
				peer.conn = conn
				pm.serverChans.peers <- peer
			}()
		case <-pm.quit:
			return
		}
	}
}
