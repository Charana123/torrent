package torrent

type peerInfo struct {
	id string
}

type choke struct {
	peerMChans          *peerMChokeChans
	peerChans           *peerChokeChans
	peerIDToPeerInfoMap map[string]peerInfo
}

func newChoke(peerMChans *peerMChokeChans, peerChans *peerChokeChans) *choke {
	return &choke{
		peerMChans: peerMChans,
		peerChans:  peerChans,
	}
}

func (c *choke) start() {
	for {
		select {
		case chokeState := <-c.peerChans.clientChokeStateChan:
			//peerInfo := c.peerIDToPeerInfoMap[chokeState.peerID]
			if chokeState.isChoked {
			} else {

			}
		case <-c.peerChans.peerHaveMessagesChan:

		}
	}
}

// How this module works
// Core - Maintains information of all peers (e.g. current outstanding pieces,
// peer bitfield, choke state) to ultimately figure out which pieces
// the client should choose to download based on piece rarity of swarm.

// Peer connections -
// -- PEER --
// choke state = if chocked clear outstanding field of peer state, if unchoked
// figure out most rare pieces and send requests (if outstanding requests isn't
// maxed)
// peer have message = send more piece requests if not maxed (based on new
// peer bitfield)
// -- DISK --
// disk piece written - figure out which peer successfully downloaded a piece,
// and send more piece requests (and the number of outstanding piece requests
// has decremented)
// -- PEER MANAGER --
// new Peer - create new peer entry
// dead Peer - remove peer entry
