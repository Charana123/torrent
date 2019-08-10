package torrent

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	utils "github.com/Charana123/go-utils"
	"github.com/marksamman/bencode"
)

type TrackerResponse struct {
	Interval int32   // interval to wait before re-requesting peer list
	Seeders  int32   // seeders	(peers that are only uploading)
	Leechers int32   // leechers (peers that are uploading and downloading) - correct ?
	Peers    []*Peer // INET Addresses of peers
}

type TrackerRequest struct {
	InfoHash   [20]byte // Urlencoded SHA1 hash of 'info' benencoded dictionary
	PeerID     [20]byte // Unique ID to locally indentify the client
	Uploaded   int64    // Bytes of torrent uploaded
	Downloaded int64    // Bytes of torrent downloaded
	Left       int64    // Bytes to download until all files of torrent are completely downloaded
	Key        int32
	Event      string      // (optinal)
	IPAddress  *net.IPAddr // (optional)
}

type TrackerTorrentScrape struct {
	InfoHash   string // Urlencoded SHA1 hash of torrent
	Complete   int
	Downloaded int
	Incomplete int
	Name       string
}

func GetPeers(trackerURLs []string, tReq *TrackerRequest) (*TrackerResponse, error) {
	for _, trackerURL := range trackerURLs {
		peerAddrs, err := queryTracker(trackerURL, tReq)
		if err != nil {
			fmt.Println(err)
			continue
		}
		return peerAddrs, nil
	}
	return nil, fmt.Errorf("None of supplied trackers responded in time")
}

func queryTracker(trackerURL string, tReq *TrackerRequest) (*TrackerResponse, error) {
	fmt.Println("trackerURL: " + trackerURL)
	if trackerURL[:6] == "udp://" {
		return queryUDPTracker(trackerURL, tReq)
	}
	if trackerURL[:7] == "http://" {
		return queryHTTPTracker(trackerURL, tReq)
	}
	return nil, fmt.Errorf("Invalid tracker URL")
}

func queryHTTPTracker(trackerURL string, tReq *TrackerRequest) (*TrackerResponse, error) {
	u, err := url.Parse(trackerURL)
	if err != nil {
		return nil, err
	}
	if !u.IsAbs() {
		return nil, fmt.Errorf("trackerURL not an absolute URL")
	}

	q := u.Query()
	urlEncodedInfoHash := url.QueryEscape(string(tReq.InfoHash[:]))
	q.Set("info_hash", urlEncodedInfoHash)
	urlEncodedPeerID := url.QueryEscape(string(tReq.PeerID[:]))
	q.Set("peer_id", urlEncodedPeerID)
	q.Set("uploaded", strconv.Itoa(int(tReq.Uploaded)))
	q.Set("downloaded", strconv.Itoa(int(tReq.Downloaded)))
	q.Set("left", strconv.Itoa(int(tReq.Left)))
	q.Set("key", strconv.Itoa(int(tReq.Key)))
	if strings.Compare(tReq.Event, "") == 0 {
	} else if strings.Compare(tReq.Event, "completed") == 0 ||
		strings.Compare(tReq.Event, "started") == 0 ||
		strings.Compare(tReq.Event, "stopped") == 0 {
		q.Set("key", tReq.Event)
	} else {
		return nil, fmt.Errorf("event string invalid")
	}
	if tReq.IPAddress != nil {
		q.Set("ip", tReq.IPAddress.String())
	}
	var peerRequestCount int32 = 50
	q.Set("numwant", strconv.Itoa(int(peerRequestCount)))
	var port int16 = 6881
	q.Set("port", strconv.Itoa(int(port)))
	q.Set("compact", "1")
	u.RawQuery = q.Encode()

	fmt.Println(u.String())
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}

	tResp := &TrackerResponse{}
	respBody, err := bencode.Decode(resp.Body)
	if failureReason, ok := respBody["failure reason"]; ok {
		return nil, fmt.Errorf(failureReason.(string))
	} else {
		if interval, ok := respBody["interval"]; ok {
			utils.RecursiveAssert(&interval, &tResp.Interval)
		}
		if complete, ok := respBody["complete"]; ok {
			utils.RecursiveAssert(&complete, &tResp.Seeders)
		}
		if incomplete, ok := respBody["incomplete"]; ok {
			utils.RecursiveAssert(&incomplete, &tResp.Leechers)
		}
		if peers, ok := respBody["peers"]; ok {
			var peersBinaryString string
			utils.RecursiveAssert(&peers, &peersBinaryString)
			peerAddrs := []byte(peersBinaryString)
			peers, err := ParsePeerAddrs(peerAddrs)
			if err != nil {
				return nil, err
			}
			tResp.Peers = peers
		}
	}
	return tResp, nil
}

// BEP 0015 - UDP Tracker Protocol for BitTorrent
func queryUDPTracker(trackerURL string, tReq *TrackerRequest) (*TrackerResponse, error) {
	trackerURLWithoutSchema := trackerURL[6:]
	connectionID, err := connectUDP(trackerURLWithoutSchema)
	if err != nil {
		return nil, err
	}
	return announceUDP(trackerURLWithoutSchema, tReq, *connectionID)
}

func connectUDP(trackerURLWithoutSchema string) (*int64, error) {
	trackerAddr, err := net.ResolveUDPAddr("udp", trackerURLWithoutSchema)
	if err != nil {
		return nil, err
	}
	trackerConn, err := net.DialUDP("udp", nil, trackerAddr)
	if err != nil {
		return nil, err
	}

	// Connection Request
	connectRequest := &bytes.Buffer{}
	protocolID, _ := hex.DecodeString("0000041727101980") // magic constant
	binary.Write(connectRequest, binary.BigEndian, protocolID)
	action := int32(0) // Connect
	binary.Write(connectRequest, binary.BigEndian, action)
	transactionID := rand.Int31()
	binary.Write(connectRequest, binary.BigEndian, transactionID)

	trackerConn.Write(connectRequest.Bytes())

	data := make([]byte, 16)
	n, err := trackerConn.Read(data)
	if err != nil {
		return nil, err
	}
	if n < 16 {
		return nil, fmt.Errorf("Malformed connection response body")
	}

	connectResponse := bytes.NewBuffer(data)

	var actionResp int32
	binary.Read(connectResponse, binary.BigEndian, &actionResp)
	if actionResp != 0 {
		return nil, fmt.Errorf("action of connection response not 'connect'")
	}

	var transactionIDResp int32
	binary.Read(connectResponse, binary.BigEndian, &transactionIDResp)
	if transactionID != transactionIDResp {
		return nil, fmt.Errorf("transactionID doesn't match")
	}

	var connectionID int64
	binary.Read(connectResponse, binary.BigEndian, &connectionID)
	return &connectionID, nil
}

func announceUDP(trackerURLWithoutSchema string, tReq *TrackerRequest, connectionID int64) (*TrackerResponse, error) {
	trackerAddr, err := net.ResolveUDPAddr("udp", trackerURLWithoutSchema)
	if err != nil {
		return nil, err
	}
	trackerConn, err := net.DialUDP("udp", nil, trackerAddr)
	if err != nil {
		return nil, err
	}

	// Connection Request
	announceRequest := &bytes.Buffer{}
	binary.Write(announceRequest, binary.BigEndian, connectionID)
	action := int32(1) // Announce
	binary.Write(announceRequest, binary.BigEndian, action)
	transactionID := rand.Int31()
	binary.Write(announceRequest, binary.BigEndian, transactionID)
	binary.Write(announceRequest, binary.BigEndian, tReq.InfoHash)
	binary.Write(announceRequest, binary.BigEndian, tReq.PeerID)
	binary.Write(announceRequest, binary.BigEndian, tReq.Downloaded)
	binary.Write(announceRequest, binary.BigEndian, tReq.Left)
	binary.Write(announceRequest, binary.BigEndian, tReq.Uploaded)
	if strings.Compare(tReq.Event, "") == 0 {
		binary.Write(announceRequest, binary.BigEndian, int32(0))
	} else if strings.Compare(tReq.Event, "completed") == 0 {
		binary.Write(announceRequest, binary.BigEndian, int32(1))
	} else if strings.Compare(tReq.Event, "started") == 0 {
		binary.Write(announceRequest, binary.BigEndian, int32(2))
	} else if strings.Compare(tReq.Event, "stopped") == 0 {
		binary.Write(announceRequest, binary.BigEndian, int32(3))
	} else {
		return nil, fmt.Errorf("event string invalid")
	}
	if tReq.IPAddress != nil {
		fmt.Println("IP: %s", tReq.IPAddress.IP)
		binary.Write(announceRequest, binary.BigEndian, tReq.IPAddress.IP)
	} else {
		binary.Write(announceRequest, binary.BigEndian, int32(0)) // defualt
	}
	binary.Write(announceRequest, binary.BigEndian, tReq.Key)
	var peerRequestCount int32 = 50
	binary.Write(announceRequest, binary.BigEndian, peerRequestCount) // Num Want (default)
	var port int16 = 6881                                             // TODO
	binary.Write(announceRequest, binary.BigEndian, port)

	trackerConn.Write(announceRequest.Bytes())

	data := make([]byte, 20+6*peerRequestCount)
	n, err := trackerConn.Read(data)
	if err != nil {
		return nil, err
	}
	if n < 20 {
		return nil, fmt.Errorf("Malformed announce response body")
	}
	announceResponse := bytes.NewBuffer(data)

	var actionResp int32
	binary.Read(announceResponse, binary.BigEndian, &actionResp)
	if actionResp != 1 {
		return nil, fmt.Errorf("action of connection response not 'announce'")
	}

	var transactionIDResp int32
	binary.Read(announceResponse, binary.BigEndian, &transactionIDResp)
	if transactionID != transactionIDResp {
		return nil, fmt.Errorf("transactionID doesn't match")
	}

	tResp := &TrackerResponse{}
	binary.Read(announceResponse, binary.BigEndian, &tResp.Interval)
	binary.Read(announceResponse, binary.BigEndian, &tResp.Leechers)
	binary.Read(announceResponse, binary.BigEndian, &tResp.Seeders)

	peerAddrs := make([]byte, announceResponse.Len())
	_, err = announceResponse.Read(peerAddrs)
	if err != nil {
		return nil, err
	}
	peers, err := ParsePeerAddrs(peerAddrs)
	if err != nil {
		return nil, err
	}
	tResp.Peers = peers

	return tResp, nil
}

func ParsePeerAddrs(peerAddrs []byte) ([]*Peer, error) {

	peers := []*Peer{}
	for i := 0; i < len(peerAddrs); i += 6 {
		ip := net.IPv4(peerAddrs[i+0], peerAddrs[i+1], peerAddrs[i+2], peerAddrs[i+3])
		port := binary.BigEndian.Uint16(peerAddrs[i+4 : i+6])
		peerAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", ip.String(), port))
		if err != nil {
			return nil, err
		}
		peers = append(peers, &Peer{
			Addr: peerAddr,
		})
	}
	return peers, nil
}

// 	scrapeURL := strings.Replazce(trackerURL, "announce", "scrape", 1)
// 	u, err := url.Parse(scrapeURL)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if !u.IsAbs() {
// 		return nil, fmt.Errorf("trackerURL not an absolute URL")
// 	}

// 	q := u.Query()
// 	q.Set("info_hash", infoHash)
// 	u.RawQuery = q.Encode()

// 	resp, err := http.Get(u.String())
// 	if err != nil {
// 		return nil, err
// 	}

// 	ttrs := []*TrackerTorrentScrape{}
// 	respBody, err := bencode.Decode(resp.Body)
// 	if files, ok := respBody["files"]; ok {
// 		var filesMap map[string]map[string]interface{}
// 		utils.RecursiveAssert(files, filesMap)

// 		for fileInfoHash, fileInfoMap := range filesMap {
// 			tts := &TrackerTorrentScrape{}
// 			tts.InfoHash = fileInfoHash
// 			if complete, ok := fileInfoMap["complete"]; ok {
// 				utils.RecursiveAssert(&complete, tts.Complete)
// 			}
// 			if downloaded, ok := fileInfoMap["downloaded"]; ok {
// 				utils.RecursiveAssert(&downloaded, tts.Downloaded)
// 			}
// 			if incomplete, ok := fileInfoMap["incomplete"]; ok {
// 				utils.RecursiveAssert(&incomplete, tts.Incomplete)
// 			}
// 			if name, ok := fileInfoMap["name"]; ok {
// 				utils.RecursiveAssert(&name, tts.Name)
// 			}
// 			ttrs = append(ttrs, tts)
// 		}
// 	}
// 	return ttrs, nil
// }
