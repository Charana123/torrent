package torrent

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
)

const (
	SNUBBED_PERIOD = 60
	CHOKE_INTERVAL = 10
	DOWNLOADERS    = 5
)

type PeerInfo struct {
	id            string
	peer          Peer
	state         connState
	lastPiece     int64
	speed         int
	shouldUnchoke bool
	snubbedClient bool
}

type choke struct {
	peerMgr PeerManager
	quit    chan int
}

func newChoke(peerMgr PeerManager, quit chan int) *choke {
	choke := &choke{
		peerMgr: peerMgr,
		quit:    quit,
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
		if time.Now().Unix()-peer.lastPiece > SNUBBED_PERIOD {
			peer.snubbedClient = true
		}
		if peer.state.peerInterested && !peer.snubbedClient {
			interested = append(interested, peer)
		} else {
			notInterested = append(notInterested, peer)
		}
	}

	// Sort in descending order of peer upload speed
	sortBySpeed(interested)
	sortBySpeed(notInterested)

	// unchoke fastest 4 uploading clients s.t. they continue to upload to the client
	// (keep the client unchoked) i.e. choose the client as one their 4 active downloaders
	speedThreshold := 0
	for i := 0; i < len(interested) && i < DOWNLOADERS-1; i++ {
		fmt.Println("interested peer id: ", interested[i].id)
		interested[i].shouldUnchoke = true
		speedThreshold = interested[i].speed
	}
	// unchoke all uninterested peers with better upload rates s.t. when they become
	// interested and start downloading from the client, they might choose the client
	// as one of their 4 active downloaders i.e. unchoke the client
	fmt.Println("speedThreshold: ", speedThreshold)
	for i := 0; i < len(notInterested) && notInterested[i].speed > speedThreshold; i++ {
		notInterested[i].shouldUnchoke = true
	}

	// optimistically unchoke a single interested peer - charity upload for peerly
	// newly connecting to the swarm
	if len(interested) > DOWNLOADERS-1 {
		interested = interested[DOWNLOADERS-1:]
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(interested), func(i, j int) {
			interested[i], interested[j] = interested[j], interested[i]
		})
		for _, peer := range interested {
			if peer.state.peerInterested {
				peer.shouldUnchoke = true
				break
			}
		}
	}

	// apply unchoke/choke
	for _, peer := range peers {
		if peer.shouldUnchoke && peer.state.clientChoking {
			peer.peer.SendUnchoke()
		}
		// keep choking and the client is currently not choking
		// then choke
		if !peer.shouldUnchoke && !peer.state.clientChoking {
			peer.peer.SendChoke()
		}
	}
}

func (c *choke) start() {

	for {
		c.choke()
		select {
		case <-c.quit:
			fmt.Println("choke stopping")
			return
		case <-time.After(time.Duration(CHOKE_INTERVAL) * time.Second):
			continue
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
