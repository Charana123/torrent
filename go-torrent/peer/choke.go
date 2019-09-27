package peer

import (
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/Charana123/torrent/go-torrent/stats"
	"github.com/Charana123/torrent/go-torrent/wire"
)

const (
	SNUBBED_PERIOD = 60
	CHOKE_INTERVAL = 10
	DOWNLOADERS    = 5
)

type PeerInfo struct {
	ID    string
	Wire  wire.Wire
	State struct {
		peerInterested   bool
		clientInterested bool
		peerChoking      bool
		clientChoking    bool
	}
	LastPiece     int64
	speed         int
	shouldUnchoke bool
	snubbedClient bool
}

type Choke interface {
	Start()
}

type choke struct {
	peerMgr PeerManager
	stats   stats.Stats
	seeding bool
	quit    chan int
}

func NewChoke(
	peerMgr PeerManager,
	stats stats.Stats,
	quit chan int) Choke {

	return &choke{
		peerMgr: peerMgr,
		stats:   stats,
		quit:    quit,
	}
}

func sortBySpeed(peers []*PeerInfo) {
	sort.Slice(peers, func(i, j int) bool {
		return peers[i].speed > peers[j].speed
	})
}

func (c *choke) choke() {
	peers := c.peerMgr.GetPeerList()
	fmt.Println("peers", peers)
	peerStats := c.stats.GetPeerStats()

	// Partition interested and uninterested peers
	interested := make([]*PeerInfo, 0)
	notInterested := make([]*PeerInfo, 0)
	for _, peer := range peers {
		if peerStat, ok := peerStats[peer.ID]; ok {
			if c.seeding {
				peer.speed = peerStat.UploadRate
			} else {
				peer.speed = peerStat.DownloadRate
			}
		}
		if peer.State.clientInterested && !peer.State.peerChoking {
			if time.Now().Unix()-peer.LastPiece > SNUBBED_PERIOD {
				peer.snubbedClient = true
			}
		}
		if peer.State.peerInterested && !peer.snubbedClient {
			interested = append(interested, peer)
		} else {
			notInterested = append(notInterested, peer)
		}
	}
	fmt.Println("interested", interested)
	fmt.Println("notInterested", notInterested)

	// Sort in descending order of peer upload speed
	sortBySpeed(interested)
	sortBySpeed(notInterested)

	// unchoke fastest 4 uploading clients s.t. they continue to upload to the client
	// (keep the client unchoked) i.e. choose the client as one their 4 active downloaders
	speedThreshold := 0
	for i := 0; i < len(interested) && i < DOWNLOADERS-1; i++ {
		interested[i].shouldUnchoke = true
		speedThreshold = interested[i].speed
	}
	// unchoke all uninterested peers with better upload rates s.t. when they become
	// interested and start downloading from the client, they might choose the client
	// as one of their 4 active downloaders i.e. unchoke the client
	for i := 0; i < len(notInterested) && notInterested[i].speed > speedThreshold; i++ {
		notInterested[i].shouldUnchoke = true
	}

	// optimistically unchoke a single interested peer - charity upload for peers
	// newly connecting to the swarm
	if len(interested) > DOWNLOADERS-1 {
		interested = interested[DOWNLOADERS-1:]
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(interested), func(i, j int) {
			interested[i], interested[j] = interested[j], interested[i]
		})
		for _, peer := range interested {
			if peer.State.peerInterested {
				peer.shouldUnchoke = true
				break
			}
		}
	}

	// apply unchoke/choke
	for _, peer := range peers {
		if peer.shouldUnchoke && peer.State.clientChoking {
			peer.Wire.SendUnchoke()
		}
		// keep choking and the client is currently not choking
		// then choke
		if !peer.shouldUnchoke && !peer.State.clientChoking {
			peer.Wire.SendChoke()
		}
	}
}

func (c *choke) Start() {

	for {
		select {
		case <-c.quit:
			return
		case <-time.After(time.Duration(CHOKE_INTERVAL * time.Second)):
			c.choke()
		}
	}
}
