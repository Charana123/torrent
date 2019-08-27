package torrent

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Charana123/torrent/go-torrent/torrent"
	bencode "github.com/jackpal/bencode-go"
)

func (tr *tracker) queryHTTPTracker(trackerURL string, event int) error {
	u, err := url.Parse(trackerURL)
	if err != nil {
		return err
	}
	if !u.IsAbs() {
		return fmt.Errorf("trackerURL not an absolute URL")
	}

	q := u.Query()
	urlEncodedInfoHash := url.QueryEscape(string(tr.torrent.InfoHash))
	q.Set("info_hash", urlEncodedInfoHash)
	urlEncodedPeerID := url.QueryEscape(string(torrent.PEER_ID))
	q.Set("peer_id", urlEncodedPeerID)
	uploaded, downloaded, left := tr.stats.GetTrackerStats()
	q.Set("uploaded", strconv.Itoa(uploaded))
	q.Set("downloaded", strconv.Itoa(downloaded))
	q.Set("left", strconv.Itoa(left))
	q.Set("key", strconv.Itoa(int(tr.key)))
	switch event {
	case COMPLETED:
		q.Set("key", "completed")
	case STARTED:
		q.Set("key", "started")
	case STOPPED:
		q.Set("key", "stopped")
	}
	if tr.clientIP != nil {
		q.Set("ip", tr.clientIP.String())
	}
	q.Set("numwant", strconv.Itoa(int(tr.numwant)))
	q.Set("port", strconv.Itoa(int(tr.serverPort)))
	q.Set("compact", "1")
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = bencode.Unmarshal(resp.Body, tr.announceResp)
	if err != nil {
		return err
	}
	if tr.announceResp.FailureReason != "" {
		return fmt.Errorf(tr.announceResp.FailureReason)
	}

	peerAddrs := []byte(tr.announceResp.Peers)
	if event != STOPPED {
		for i := 0; i < len(peerAddrs); i += 6 {
			ip := net.IPv4(peerAddrs[i+0], peerAddrs[i+1], peerAddrs[i+2], peerAddrs[i+3])
			port := binary.BigEndian.Uint16(peerAddrs[i+4 : i+6])
			id := fmt.Sprintf("%s:%d", ip.String(), port)
			tr.peerMgr.AddPeer(id, nil)
		}
	}
	return nil
}
