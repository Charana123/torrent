package tracker

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/Charana123/torrent/go-torrent/peer"
	"github.com/Charana123/torrent/go-torrent/stats"
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
	announceList [][]string
	infoHash     []byte
	peerMgr      peer.PeerManager
	stats        stats.Stats
	quit         chan int
	serverPort   int
	key          int32
	numwant      int32

	interval      int32
	totalLeechers int32 `bencode:"incomplete"`
	totalSeeders  int32 `bencode:"complete"`
}

func genKey() int32 {
	rand.Seed(time.Now().Unix())
	return rand.Int31()
}

func NewTracker(
	announceList [][]string,
	infoHash []byte,
	stats stats.Stats,
	peerMgr peer.PeerManager,
	quit chan int,
	serverPort int) Tracker {

	tr := &tracker{
		announceList: announceList,
		infoHash:     infoHash,
		quit:         quit,
		serverPort:   serverPort,
		peerMgr:      peerMgr,
		key:          genKey(),
		numwant:      -1,
		stats:        stats,
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
	for _, trackerURLs := range tr.announceList {
		for _, trackerURL := range trackerURLs {
			fmt.Println("querying tracker: ", trackerURL)
			tr.queryTracker(trackerURL, event)
		}
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
			tr.peerMgr.NewInterval()
		}
	}
}
