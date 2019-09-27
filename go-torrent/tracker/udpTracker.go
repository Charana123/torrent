package tracker

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/Charana123/torrent/go-torrent/torrent"
)

// BEP 0015 - UDP Tracker Protocol for BitTorrent
func (tr *tracker) queryUDPTracker(trackerURL string, event int) error {
	udpAddress := trackerURL[6:]
	udpAddress = strings.TrimSuffix(udpAddress, "/announce")
	trackerAddr, err := net.ResolveUDPAddr("udp", udpAddress)
	if err != nil {
		return err
	}
	trackerConn, err := net.DialUDP("udp", nil, trackerAddr)
	if err != nil {
		return err
	}

	connectionID, err := tr.connectUDP(trackerConn)
	if err != nil {
		return err
	}
	return tr.announceUDP(trackerConn, event, connectionID)
}

func (tr *tracker) connectUDP(trackerConn *net.UDPConn) (int64, error) {

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
	n, err := io.ReadFull(trackerConn, data)
	if err != nil {
		return 0, err
	}
	if n < 16 {
		return 0, fmt.Errorf("Malformed connection response body")
	}

	connectResponse := bytes.NewBuffer(data)

	var actionResp int32
	binary.Read(connectResponse, binary.BigEndian, &actionResp)
	if actionResp != 0 {
		return 0, fmt.Errorf("action of connection response not 'connect'")
	}

	var transactionIDResp int32
	binary.Read(connectResponse, binary.BigEndian, &transactionIDResp)
	if transactionID != transactionIDResp {
		return 0, fmt.Errorf("transactionID doesn't match")
	}

	var connectionID int64
	binary.Read(connectResponse, binary.BigEndian, &connectionID)
	return connectionID, nil
}

func (tr *tracker) announceUDP(trackerConn *net.UDPConn, event int, connectionID int64) error {

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
	binary.Write(announceRequest, binary.BigEndian, int64(downloaded))
	binary.Write(announceRequest, binary.BigEndian, int64(left))
	binary.Write(announceRequest, binary.BigEndian, int64(uploaded))
	binary.Write(announceRequest, binary.BigEndian, int32(event))
	binary.Write(announceRequest, binary.BigEndian, int32(0)) // defualt
	binary.Write(announceRequest, binary.BigEndian, tr.key)
	binary.Write(announceRequest, binary.BigEndian, int32(-1))
	binary.Write(announceRequest, binary.BigEndian, uint16(tr.serverPort))

	trackerConn.Write(announceRequest.Bytes())

	announceResponse := &bytes.Buffer{}
	trackerConn.SetDeadline(time.Now().Add(time.Second * 2))
	n, err := io.Copy(announceResponse, trackerConn)
	if err, ok := err.(net.Error); !ok || !err.Timeout() {
		return err
	}
	if n < 20 {
		return fmt.Errorf("Malformed announce response body")
	}

	// announceResponse := bytes.NewBuffer(data)
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
			port := binary.BigEndian.Uint16([]byte(peerAddrs[i+4 : i+6]))
			id := fmt.Sprintf("%s:%d", ip, port)
			tr.peerMgr.AddPeer(id, nil)
		}
	}
	return nil
}
