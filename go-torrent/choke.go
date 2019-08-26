package torrent

import (
	"fmt"
	"sort"
	"time"
)

const (
	CHOKE_INTERVAL = 10
	DOWNLOADERS    = 4
)

type PeerInfo struct {
	id    string
	state struct {
		peerInterested   bool
		clientInterested bool
		peerChoking      bool
		clientChoking    bool
	}
	speed         int
	shouldUnchoke bool
}

type choke struct {
	peerMgr PeerManager
	quit    chan int
}

func newChoke(quit chan int) *choke {
	choke := &choke{
		quit: quit,
	}
	go choke.start()
	return choke
}

func sortBySpeed(peers []*PeerInfo) {
	sort.Slice(peers, func(i, j int) bool {
		return peers[i].speed > peers[j].speed
	})
}

func (c *choke) choke() {
	peers := c.peerMgr.GetPeerList()

	// Partition interested and uninterested peers
	interested := make([]*PeerInfo, 0)
	notInterested := make([]*PeerInfo, 0)
	for _, peer := range peers {
		if peer.state.peerInterested {
			interested = append(interested, peer)
		} else {
			notInterested = append(notInterested, peer)
		}
	}

	// Sort in descending order of peer upload speed
	sortBySpeed(interested)
	sortBySpeed(notInterested)

	// maintain (DOWNLOADERS-1) downloaders - interested and unchoked peers
	speedThreshold := 0
	for i := 0; i < len(interested) && i < DOWNLOADERS-1; i++ {
		interested[i].shouldUnchoke = true
		speedThreshold = interested[i].speed
	}
	// unchoke all uninterested peers with better upload rates
	for i := 0; i < len(notInterested) && notInterested[i].speed > speedThreshold; i++ {
		notInterested[i].shouldUnchoke = true
	}

	// choke/unchoke peers
	for _, peer := range peers {
		if peer.shouldUnchoke && peer.state.clientChoking {
			// unchoke peer
		}
		if !peer.shouldUnchoke && !peer.state.clientChoking {
			// choke peer
		}
	}
}

func (c *choke) start() {

	for {
		select {
		case <-c.quit:
			fmt.Println("choke stopping")
			return
		case <-time.After(time.Duration(CHOKE_INTERVAL) * time.Second):
			c.choke()
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
