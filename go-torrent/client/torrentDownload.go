package client

import (
	"bytes"
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

type TorrentStats struct {
	Peers   int
	Seeders int
}

type TorrentDownload interface {
	// Start() error
	// Stop()
	// RemoveTorrent()
	// RemoveTorrentAndData()
	// VerifyData()
	// GetFiles() []*FileDownload
	// GetInfoHash() []byte
	// Size() int
	// Name() string
	// NumPieces() int
	// GetStats() TorrentStats
}

type torrentDownload struct {
	stopped  bool
	quit     chan int
	peerMgr  peer.PeerManager
	storage  storage.Storage
	pieceMgr piece.PieceManager
	stats    stats.Stats
	tor      *torrent.Torrent
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

func NewTorrentDownload(tor *torrent.Torrent) TorrentDownload {
	return &torrentDownload{
		tor: tor,
	}
}

// Start/Resume downloading/uploading torrent
func (d *torrentDownload) Start() error {

	quit := make(chan int)
	d.quit = quit

	d.storage = storage.NewRandomAccessStorage(d.tor)
	d.storage.Init()
	clientBitfield, _, left := d.storage.GetCurrentDownloadState()
	d.stats = stats.NewStats(0, 0, left)
	d.pieceMgr = piece.NewRarestFirstPieceManager(d.tor, d.storage, clientBitfield)
	d.peerMgr = peer.NewPeerManager(d.tor, d.pieceMgr, d.storage, d.stats)
	choke := peer.NewChoke(d.tor, d.peerMgr, d.pieceMgr, d.stats, quit)
	go choke.Start()
	sv, err := server.NewServer(d.peerMgr, quit)
	if err != nil {
		return err
	}
	go sv.Serve()
	tracker := tracker.NewTracker(d.tor, d.stats, d.peerMgr, quit, sv.GetServerPort())
	go tracker.Start()
	return nil
}

// Stop downloading/uploading torrent
func (d *torrentDownload) Stop() {
	close(d.quit)
	go d.peerMgr.StopPeers()
}

func (d *torrentDownload) VerifyData() {

}

func (d *torrentDownload) GetFiles() []*FileDownload {
	return nil
}

func (d *torrentDownload) GetInfoHash() []byte {
	return nil
}

func (d *torrentDownload) Size() int {
	return 0
}

func (d *torrentDownload) Name() string {
	return ""
}

func (d *torrentDownload) NumPieces() int {
	return 0
}
