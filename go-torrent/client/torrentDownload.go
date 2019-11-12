package client

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

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

func parseMagnetURI(magnetURI string) (*torrent.MagnetURI, error) {
	r1, _ := regexp.Compile(`magnet:\?xt=urn:(\S{4}):(\S{40})`)
	g1 := r1.FindStringSubmatch(magnetURI)
	if len(g1) == 0 {
		return nil, fmt.Errorf("Malformed magnet URI")
	}
	if g1[1] == "btmh" {
		return nil, fmt.Errorf("Client doesn't support multihash format")
	}
	muri := &torrent.MagnetURI{}
	muri.InfoHashHex = g1[2]
	r2, _ := regexp.Compile(`&(\S*?)=(\S*?)(?=(?:&|$))`)
	g2 := r2.FindAllStringSubmatch(magnetURI, -1)
	for i := 0; i < len(g2); i++ {
		if g2[i][1] == "name" {
			muri.Name = g2[i][2]
		}
		if g2[i][1] == "tr" {
			muri.Trackers = append(muri.Trackers, g2[i][2])
		}
		if g2[i][1] == "x.pe" {
			muri.Peers = append(muri.Peers, g2[i][2])
		}
	}
	return muri, nil
}

func NewTorrentFromMagnet(magnetURI string) (TorrentDownload, error) {
	muri, err := parseMagnetURI(magnetURI)
	if err != nil {
		return nil, err
	}
	return &torrentDownload{
		muri: muri,
	}, nil
}

func NewTorrentDownload(tor *torrent.Torrent, dataDirectory string) TorrentDownload {
	return &torrentDownload{
		tor:           tor,
		dataDirectory: dataDirectory,
	}
}

func (d *torrentDownload) metadataExchange() {
	// tr := &torrent.Torrent{
	// 	InfoHash: []byte(d.muri.infoHashHex),
	// 	MetaInfo: torrent.MetaInfo{
	// 		AnnounceList: [][]string{d.muri.trackers},
	// 	},
	// }

	// tracker.NewTracker(tr, nil)
	// implement some way to insert
}

// Start/Resume downloading/uploading torrent
func (d *torrentDownload) Start() error {

	quit := make(chan int)
	d.quit = quit

	d.storage = storage.NewRandomAccessStorage(d.dataDirectory)
	clientBitfield, _, left := d.storage.GetCurrentDownloadState()
	d.stats = stats.NewStats(0, 0, left)
	d.pieceMgr = piece.NewRarestFirstPieceManager(d.storage, clientBitfield)
	d.peerMgr = peer.NewPeerManager(d.tor, d.muri, d.pieceMgr, d.storage, d.stats)
	choke := peer.NewChoke(d.peerMgr, d.pieceMgr, d.stats, quit)
	sv, err := server.NewServer(d.peerMgr, quit)
	if err != nil {
		return err
	}

	// tracker
	announceList := map[bool][][]string{
		true: map[bool][][]string{
			true:  d.tor.MetaInfo.AnnounceList,
			false: [][]string{[]string{d.tor.MetaInfo.Announce}}}[len(d.tor.MetaInfo.AnnounceList) > 0],
		false: [][]string{d.muri.Trackers}}[d.tor != nil]
	infoHash := map[bool][]byte{
		true:  d.tor.InfoHash,
		false: []byte(d.muri.InfoHashHex)}[d.tor != nil]
	tracker := tracker.NewTracker(announceList, infoHash, d.stats, d.peerMgr, quit, sv.GetServerPort())
	go tracker.Start()

	go func() {
		// d.storage.Init(d.tor)
		// d.pieceMgr.Init(d.tor)
		// d.peerMgr.Init(d.tor)
		go choke.Start(d.tor)
		// go sv.Serve(d.tor)
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
