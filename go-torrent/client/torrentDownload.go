package client

import (
	"bytes"
	"encoding/hex"
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

type TorrentStats struct {
	Peers   int
	Seeders int
}

type TorrentDownload interface {
	Start() error
	Stop()
	VerifyData()
	// GetFiles() []*FileDownload
	GetInfoHash() []byte
	// Size() int
	// Name() string
	// NumPieces() int
	// GetStats() TorrentStats
}

type torrentDownload struct {
	stopping      bool
	stopped       bool
	quit          chan int
	peerMgr       peer.PeerManager
	storage       storage.Storage
	pieceMgr      piece.PieceManager
	stats         stats.Stats
	dataDirectory string
	tor           *torrent.Torrent
	muri          *torrent.MagnetURI
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

func NewTorrentFromMagnet(muri *torrent.MagnetURI) TorrentDownload {
	return &torrentDownload{
		muri: muri,
	}
}

func NewTorrentDownload(tor *torrent.Torrent, dataDirectory string) TorrentDownload {
	return &torrentDownload{
		tor:           tor,
		dataDirectory: dataDirectory,
	}
}

// Start/Resume downloading/uploading torrent
func (d *torrentDownload) Start() error {

	quit := make(chan int)
	d.quit = quit

	d.storage = storage.NewRandomAccessStorage(d.dataDirectory)
	d.stats = stats.NewStats(0, 0, 0)
	d.pieceMgr = piece.NewRarestFirstPieceManager(d.storage)
	mdMgr, downloadedChan := piece.NewMetadataManager(d.muri)
	d.peerMgr = peer.NewPeerManager(d.tor, d.pieceMgr, mdMgr, d.storage, d.stats)
	choke := peer.NewChoke(d.peerMgr, d.pieceMgr, d.stats, quit)
	sv, err := server.NewServer(d.peerMgr, quit)
	if err != nil {
		return err
	}

	// tracker
	var announceList [][]string
	var infoHash []byte
	if d.tor != nil {
		if len(d.tor.MetaInfo.AnnounceList) > 0 {
			announceList = d.tor.MetaInfo.AnnounceList
		} else {
			announceList = [][]string{[]string{d.tor.MetaInfo.Announce}}
		}
	} else {
		announceList = [][]string{d.muri.Trackers}
	}
	if d.tor != nil {
		infoHash = d.tor.InfoHash
	} else {
		infoHash, err = hex.DecodeString(d.muri.InfoHashHex)
	}
	tracker := tracker.NewTracker(announceList, infoHash, d.stats, d.peerMgr, quit, sv.GetServerPort())
	go tracker.Start()

	go func() {
		if d.tor == nil {
			d.tor = <-downloadedChan
			fmt.Println("Metadata Downloaded")
		}
		d.storage.Init(d.tor)
		clientBitfield, _, _ := d.storage.GetCurrentDownloadState()
		d.pieceMgr.Init(d.tor, clientBitfield)
		d.peerMgr.Init(d.tor)
		go choke.Start(d.tor)
		go sv.Serve()
	}()

	return nil
}

// Stop downloading/uploading torrent
func (d *torrentDownload) Stop() {
	close(d.quit)
	go d.peerMgr.StopPeers()
}

// Used to remove corrupted pieces while a torrent is downloading/seeding
func (d *torrentDownload) VerifyData() {
	bitfield, _, _ := d.storage.GetCurrentDownloadState()
	d.pieceMgr.VerifyBitField(bitfield)
}

func (d *torrentDownload) GetFiles() []*FileDownload {
	return nil
}

func (d *torrentDownload) GetInfoHash() []byte {
	return d.tor.InfoHash
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
