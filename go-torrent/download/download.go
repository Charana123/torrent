package download

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Charana123/torrent/go-torrent/piece"
	"github.com/Charana123/torrent/go-torrent/server"
	"github.com/Charana123/torrent/go-torrent/stats"
	"github.com/Charana123/torrent/go-torrent/storage"
	"github.com/Charana123/torrent/go-torrent/tracker"

	"github.com/Charana123/torrent/go-torrent/peer"
	"github.com/Charana123/torrent/go-torrent/torrent"
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

	fmt.Println(t)

	storage := storage.NewRandomAccessStorage(t)
	storage.Init()
	clientBitfield, _, left := storage.GetCurrentDownloadState()
	stats := stats.NewStats(0, 0, left)
	pieceMgr := piece.NewRarestFirstPieceManager(t, storage, clientBitfield)
	d.peerMgr = peer.NewPeerManager(t, pieceMgr, storage, stats)
	choke := peer.NewChoke(t, d.peerMgr, pieceMgr, stats, quit)
	go choke.Start()
	sv, err := server.NewServer(d.peerMgr, quit)
	if err != nil {
		return err
	}
	go sv.Serve()
	tracker := tracker.NewTracker(t, stats, d.peerMgr, quit, sv.GetServerPort())
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
