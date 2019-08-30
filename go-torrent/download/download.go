package client

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/Charana123/torrent/go-torrent/piece"
	"github.com/Charana123/torrent/go-torrent/stats"

	"github.com/Charana123/torrent/go-torrent/storage"

	"github.com/Charana123/torrent/go-torrent/peer"
	"github.com/Charana123/torrent/go-torrent/server"
	"github.com/Charana123/torrent/go-torrent/torrent"
	"github.com/Charana123/torrent/go-torrent/tracker"
)

type Download interface {
	Start(path string) error
	Stop()
	CleanUp()
}

type download struct {
	quit    chan int
	peerMgr peer.PeerManager
}

func getExternalIP() (string, error) {
	rsp, err := http.Get("http://checkip.amazonaws.com")
	if err != nil {
		return "", err
	}
	defer rsp.Body.Close()

	buf, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return "", err
	}

	return string(bytes.TrimSpace(buf)), nil
}

func NewDownload() Download {
	return &download{}
}

// Start/Resume downloading/uploading torrent
func (d *download) Start(path string) error {

	t, err := torrent.NewTorrent(path)
	if err != nil {
		return err
	}

	quit := make(chan int)
	d.quit = quit

	storage := storage.NewRandomAccessStorage(t)
	go storage.Init()
	clientBitfield, _, left := storage.GetCurrentDownloadState()
	stats := stats.NewStats(0, 0, left)
	pieceMgr := piece.NewRarestFirstPieceManager(t, clientBitfield)
	d.peerMgr = peer.NewPeerManager(t, pieceMgr, storage, stats)
	sv, err := server.NewServer(d.peerMgr, quit)
	go sv.Serve()
	if err != nil {
		return err
	}
	clientIP, err := getExternalIP()
	if err != nil {
		return err
	}
	tracker := tracker.NewTracker(t, quit, sv.GetServerPort(), net.ParseIP(clientIP), stats)
	go tracker.Start()
	return nil
}

// Stop downloading/uploading torrent
func (d *download) Stop() {
	close(d.quit)
	go d.peerMgr.StopPeers()
}

// Delete  (potentially only partially) downloaded torrent files
func (d *download) CleanUp() {

}

// TODO - verify which pieces have already been downloaded and verified
