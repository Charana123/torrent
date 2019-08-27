package torrent

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/Charana123/torrent/go-torrent/peer"

	"github.com/Charana123/torrent/go-torrent/torrent"
)

const (
	NONE      = 0
	COMPLETED = 1
	STARTED   = 2
	STOPPED   = 3
)

type Tracker interface {
}

type tracker struct {
	torrent      *torrent.Torrent
	peerMgr      peer.PeerManager
	quit         chan int
	port         uint16
	ip           *net.IP
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

func newTracker(
	torrent *torrent.Torrent,
	quit chan int,
	port int,
	ip *net.IP) *tracker {

	// TODO - DI for `tStatsEvent` and `tCompleteEventChan`
	// TODO - Pass `tStatsResponse` to whoever
	return &tracker{
		torrent: torrent,
		quit:    quit,
		port:    uint16(port),
		ip:      ip,
		key:     genKey(),
		numwant: 50,
	}
}

func (tr *tracker) announceTracker(trackerURL string) error {

	var queryTracker func(string, int) error
	if trackerURL[:6] == "udp://" {
		queryTracker = tr.queryUDPTracker
	} else if trackerURL[:7] == "http://" {
		queryTracker = tr.queryHTTPTracker
	} else {
		return fmt.Errorf("Invalid schema for trackerURL")
	}

	queryTracker(trackerURL, STARTED)
	intervalD := time.Duration(tr.announceResp.Interval)

	for {
		var err error
		select {
		case <-tr.quit:
			log.Println("Safely terminating tracker")
			queryTracker(trackerURL, STOPPED)
			return nil
		case <-time.After(intervalD):
			err = queryTracker(trackerURL, NONE)
			// case <-tr.tStatsEvent:
			// 	tr.tStatsResponse <- &tStats{
			// 		leechers: tr.announceResp.Leechers,
			// 		seeders:  tr.announceResp.Seeders,
			// 	}
			// case <-tr.tCompleteEventChan:
			// 	err = queryTracker(trackerURL, COMPLETED)
		}
		if err != nil {
			return nil
		}
	}
}

func (tr *tracker) connectTracker() {
	if len(tr.torrent.MetaInfo.AnnounceList) > 0 {
		for _, trackerURLs := range tr.torrent.MetaInfo.AnnounceList {
			for _, trackerURL := range trackerURLs {
				err := tr.announceTracker(trackerURL)
				if err == nil {
					// We've successfully connected and disconnected
					return
				}
				// Otherwise, lower tracker priority for its tier
				trackerURLs = append(
					trackerURLs[:len(trackerURLs)-1],
					trackerURL)
			}
		}
	} else {
		tr.announceTracker(tr.torrent.MetaInfo.Announce)
	}
}

func (tr *tracker) start() {
	for {
		select {
		case <-tr.quit:
			return
		// Connect or Reconnect if channel not closed
		default:
			tr.connectTracker()
		}
	}
}
