package tracker

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/Charana123/torrent/go-torrent/peer"
	"github.com/Charana123/torrent/go-torrent/stats"

	"github.com/Charana123/torrent/go-torrent/torrent"
)

const (
	NONE      = 0
	COMPLETED = 1
	STARTED   = 2
	STOPPED   = 3
)

type Tracker interface {
	Start()
}

type tracker struct {
	torrent    *torrent.Torrent
	peerMgr    peer.PeerManager
	stats      stats.Stats
	quit       chan int
	serverPort int
	key        int32
	numwant    int32

	interval      int32
	totalLeechers int32 `bencode:"incomplete"`
	totalSeeders  int32 `bencode:"complete"`
}

func genKey() int32 {
	rand.Seed(time.Now().Unix())
	return rand.Int31()
}

func NewTracker(
	torrent *torrent.Torrent,
	stats stats.Stats,
	peerMgr peer.PeerManager,
	quit chan int,
	serverPort int) Tracker {

	tr := &tracker{
		torrent:    torrent,
		quit:       quit,
		serverPort: serverPort,
		peerMgr:    peerMgr,
		key:        genKey(),
		numwant:    -1,
		stats:      stats,
	}
	return tr
}

func (tr *tracker) queryTracker(trackerURL string, event int) error {
	var qt func(string, int) error
	if trackerURL[:6] == "udp://" {
		qt = tr.queryUDPTracker
	} else if trackerURL[:7] == "http://" {
		qt = tr.queryHTTPTracker
	} else {
		return fmt.Errorf("Invalid schema for trackerURL")
	}
	err := qt(trackerURL, event)
	return err
}

func (tr *tracker) queryTrackers(event int) {
	if len(tr.torrent.MetaInfo.AnnounceList) > 0 {
		for _, trackerURLs := range tr.torrent.MetaInfo.AnnounceList {
			for _, trackerURL := range trackerURLs {
				fmt.Println("querying tracker: ", trackerURL)
				tr.queryTracker(trackerURL, event)
			}
		}
	} else {
		tr.queryTracker(tr.torrent.MetaInfo.Announce, event)
	}
}

func (tr *tracker) Start() {
	for {
		select {
		case <-tr.quit:
			tr.queryTrackers(STOPPED)
			return
		case <-time.After(time.Second * time.Duration(tr.interval)):
			tr.queryTrackers(NONE)
			fmt.Println("interval: ", tr.interval)
		}
	}
}
