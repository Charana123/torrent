package peer

import (
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/Charana123/torrent/go-torrent/torrent"

	"github.com/Charana123/torrent/go-torrent/piece"

	"github.com/Charana123/torrent/go-torrent/stats"
)

const (
	SNUBBED_PERIOD = 60
	CHOKE_INTERVAL = 10
	DOWNLOADERS    = 5
)

type PeerInfo struct {
	ID    string
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
	torrent  *torrent.Torrent
	peerMgr  PeerManager
	pieceMgr piece.PieceManager
	stats    stats.Stats
	seeding  bool
	quit     chan int
}

func NewChoke(
	peerMgr PeerManager,
	pieceMgr piece.PieceManager,
	stats stats.Stats,
	quit chan int) Choke {

	return &choke{
		peerMgr:  peerMgr,
		pieceMgr: pieceMgr,
		stats:    stats,
		quit:     quit,
	}
}

func sortBySpeed(peers []*PeerInfo) {
	sort.Slice(peers, func(i, j int) bool {
		return peers[i].speed > peers[j].speed
	})
}

func (c *choke) choke() {
	peers := c.peerMgr.GetPeerList()

	peerInfos := []*PeerInfo{}
	for _, peer := range peers {
		id, state, lastPiece := peer.GetPeerInfo()
		peerInfo := &PeerInfo{
			ID:        id,
			State:     state,
			LastPiece: lastPiece,
		}
		peerInfos = append(peerInfos, peerInfo)
	}
	peerStats := c.stats.GetPeerStats()

	// Partition interested and uninterested peers
	interested := make([]*PeerInfo, 0)
	notInterested := make([]*PeerInfo, 0)
	for _, peerInfo := range peerInfos {
		if peerStat, ok := peerStats[peerInfo.ID]; ok {
			if c.seeding {
				peerInfo.speed = peerStat.UploadRate
			} else {
				peerInfo.speed = peerStat.DownloadRate
			}
		}
		if peerInfo.State.clientInterested && !peerInfo.State.peerChoking {
			if time.Now().Unix()-peerInfo.LastPiece > SNUBBED_PERIOD {
				peerInfo.snubbedClient = true
			}
		}
		if peerInfo.State.peerInterested && !peerInfo.snubbedClient {
			interested = append(interested, peerInfo)
		} else {
			notInterested = append(notInterested, peerInfo)
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
	for i, peerInfo := range peerInfos {
		if peerInfo.shouldUnchoke && peerInfo.State.clientChoking {
			fmt.Print(peerInfo.ID, "unchoke")
			peers[i].SendUnchoke()
		}
		if !peerInfo.shouldUnchoke && !peerInfo.State.clientChoking {
			fmt.Print(peerInfo.ID, "choke")
			peers[i].SendChoke()
		}
	}
}

func (c *choke) PrintDownloadPercentage() {
	percentage := (float32(c.pieceMgr.GetPiecesDownloaded()) / float32(c.torrent.NumPieces)) * 100
	fmt.Println("download percentage: ", percentage)
}

func (c *choke) Start(tor *torrent.Torrent) {
	c.torrent = tor

	for {
		select {
		case <-c.quit:
			return
		case <-time.After(time.Duration(CHOKE_INTERVAL * time.Second)):
			c.PrintDownloadPercentage()
			c.choke()
		}
	}
}
