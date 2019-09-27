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
	torrent      *torrent.Torrent
	peerMgr      peer.PeerManager
	stats        stats.Stats
	quit         chan int
	serverPort   int
	key          int32
	numwant      int32
	announceResp struct {
		FailureReason string `bencode:"failure reason"`
		Interval      int32
		Leechers      int32 `bencode:"incomplete"`
		Seeders       int32 `bencode:"complete"`
		Peers         string
	}
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
		numwant:    50,
		stats:      stats,
	}
	return tr
}

func (tr *tracker) announceTracker(trackerURL string) error {
	fmt.Println("announceTracker", trackerURL)

	var queryTracker func(string, int) error
	if trackerURL[:6] == "udp://" {
		queryTracker = tr.queryUDPTracker
	} else if trackerURL[:7] == "http://" {
		queryTracker = tr.queryHTTPTracker
	} else {
		return fmt.Errorf("Invalid schema for trackerURL")
	}

	for {
		fmt.Println("tr.announceResp.Interval", tr.announceResp.Interval)
		select {
		case <-tr.quit:
			queryTracker(trackerURL, STOPPED)
			return nil
		case <-time.After(time.Second * time.Duration(tr.announceResp.Interval)):
			err := queryTracker(trackerURL, NONE)
			fmt.Println(err)
			if err != nil {
				return err
			}
		}
	}
}

func (tr *tracker) Start() {
	for {
		if len(tr.torrent.MetaInfo.AnnounceList) > 0 {
			for _, trackerURLs := range tr.torrent.MetaInfo.AnnounceList {
				for i, trackerURL := range trackerURLs {
					err := tr.announceTracker(trackerURL)
					// fmt.Println("error", err)
					// tracker must stop
					if err == nil {
						return
					}
					// Otherwise, lower tracker priority for its tier
					trackerURLs = append(append(trackerURLs[i:], trackerURLs[:i]...), trackerURL)
				}
			}
		} else {
			tr.announceTracker(tr.torrent.MetaInfo.Announce)
			// Wait a second before trying to re-connect to SAME tracker
			<-time.After(time.Second)
		}
	}
}
