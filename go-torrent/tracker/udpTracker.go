package torrent

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"

	"github.com/Charana123/torrent/go-torrent/torrent"
)

// BEP 0015 - UDP Tracker Protocol for BitTorrent
func (tr *tracker) queryUDPTracker(trackerURL string, event int) error {
	trackerURLWithoutSchema := trackerURL[6:]
	connectionID, err := tr.connectUDP(trackerURLWithoutSchema)
	if err != nil {
		return err
	}
	return tr.announceUDP(trackerURLWithoutSchema, event, *connectionID)
}

func (tr *tracker) connectUDP(trackerURLWithoutSchema string) (*int64, error) {
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

func (tr *tracker) announceUDP(trackerURLWithoutSchema string, event int, connectionID int64) error {
	trackerAddr, err := net.ResolveUDPAddr("udp", trackerURLWithoutSchema)
	if err != nil {
		return err
	}
	trackerConn, err := net.DialUDP("udp", nil, trackerAddr)
	if err != nil {
		return err
	}

	// Connection Request
	announceRequest := &bytes.Buffer{}
	binary.Write(announceRequest, binary.BigEndian, connectionID)
	action := int32(1) // Announce
	binary.Write(announceRequest, binary.BigEndian, action)
	transactionID := rand.Int31()
	binary.Write(announceRequest, binary.BigEndian, transactionID)
	binary.Write(announceRequest, binary.BigEndian, tr.torrent.InfoHash)
	binary.Write(announceRequest, binary.BigEndian, torrent.PEER_ID)
	uploaded, downloaded, left := tr.stats.GetTrackerStats()
	binary.Write(announceRequest, binary.BigEndian, downloaded)
	binary.Write(announceRequest, binary.BigEndian, left)
	binary.Write(announceRequest, binary.BigEndian, uploaded)
	binary.Write(announceRequest, binary.BigEndian, event)
	if tr.clientIP != nil {
		binary.Write(announceRequest, binary.BigEndian, tr.clientIP)
	} else {
		binary.Write(announceRequest, binary.BigEndian, int32(0)) // defualt
	}
	binary.Write(announceRequest, binary.BigEndian, tr.key)
	binary.Write(announceRequest, binary.BigEndian, tr.numwant)
	binary.Write(announceRequest, binary.BigEndian, tr.serverPort)

	trackerConn.Write(announceRequest.Bytes())

	data, err := ioutil.ReadAll(trackerConn)
	if err != nil {
		return err
	}
	if len(data) < 20 {
		return fmt.Errorf("Malformed announce response body")
	}

	announceResponse := bytes.NewBuffer(data)
	var actionResp int32
	binary.Read(announceResponse, binary.BigEndian, &actionResp)
	if actionResp != 1 {
		return fmt.Errorf("action of connection response not 'announce'")
	}
	var transactionIDResp int32
	binary.Read(announceResponse, binary.BigEndian, &transactionIDResp)
	if transactionID != transactionIDResp {
		return fmt.Errorf("transactionID doesn't match")
	}
	binary.Read(announceResponse, binary.BigEndian, &tr.announceResp.Interval)
	binary.Read(announceResponse, binary.BigEndian, &tr.announceResp.Leechers)
	binary.Read(announceResponse, binary.BigEndian, &tr.announceResp.Seeders)

	peerAddrs, err := ioutil.ReadAll(announceResponse)
	if err != nil {
		return err
	}

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
