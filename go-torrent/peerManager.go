package torrent

import (
	"net"

	bitmap "github.com/boljen/go-bitmap"
)

type peerMChokeChans struct {
	newPeer chan *peer
}

type peerManager struct {
	torrent        *Torrent
	serverChans    *serverPeerMChans
	trackerChans   *trackerPeerMChans
	chokeChans     *peerMChokeChans
	chokePeerChans *peerChokeChans
	// diskPeerChans  *peerDiskChans
	peers    map[string]*peer
	numPeers int
	maxPeers int
}

func newPeerManager(
	torrent *Torrent,
	serverChans *serverPeerMChans,
	trackerChans *trackerPeerMChans) *peerManager {

	pm := &peerManager{
		torrent:      torrent,
		serverChans:  serverChans,
		trackerChans: trackerChans,
		chokeChans: &peerMChokeChans{
			newPeer: make(chan *peer),
		},
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

			fromChokeChans := &chokePeerChans{}
			peer.toChokeChans = pm.chokePeerChans
			peer.fromChokeChans = fromChokeChans
			peer.torrent = pm.torrent
			peer.bitfield = bitmap.New(pm.torrent.numPieces)
			go func() { pm.chokeChans.newPeer <- fromChokeChans }()

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
		}

	}
}
