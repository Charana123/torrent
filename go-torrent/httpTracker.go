package torrent

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"

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
	urlEncodedInfoHash := url.QueryEscape(string(tr.torrent.infoHash))
	q.Set("info_hash", urlEncodedInfoHash)
	urlEncodedPeerID := url.QueryEscape(string(PEER_ID))
	q.Set("peer_id", urlEncodedPeerID)
	q.Set("uploaded", strconv.Itoa(int(tr.stats.uploaded)))
	q.Set("downloaded", strconv.Itoa(int(tr.stats.downloaded)))
	q.Set("left", strconv.Itoa(int(tr.stats.left)))
	q.Set("key", strconv.Itoa(int(tr.key)))
	switch event {
	case COMPLETED:
		q.Set("key", "completed")
	case STARTED:
		q.Set("key", "started")
	case STOPPED:
		q.Set("key", "stopped")
	}
	if tr.ip != nil {
		q.Set("ip", tr.ip.String())
	}
	q.Set("numwant", strconv.Itoa(int(tr.numwant)))
	q.Set("port", strconv.Itoa(int(tr.port)))
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
			peer := &peer{}
			peer.id = fmt.Sprintf("%s:%d", ip.String(), port)
			tr.peerMChans.peers <- peer
		}
	}
	return nil
}
