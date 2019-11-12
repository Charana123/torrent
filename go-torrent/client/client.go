package client

import (
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"

	"github.com/Charana123/torrent/go-torrent/torrent"
)

type Client interface {
	AddTorrent(torrentReader io.ReadSeeker) TorrentDownload
	AddMagnet(magnetURI string) (TorrentDownload, error)
	RemoveTorrent(infoHashHex string)
	RemoveTorrentAndData(infoHashHex string)
	GetTorrents() []TorrentDownload

	// StopTorrent(torrentID string)
	// StopFile(torrentID string, fileIndex int)
	// StartFile(torrentID string, fileIndex int)
}

type client struct {
	// torrentDownloads map[string]TorrentDownload
	torrents     []TorrentDownload
	torrentsPath string
	dataPath     string
}

func NewClient(storagePath string) Client {
	c := &client{
		torrentsPath: storagePath + "/torrent",
		dataPath:     storagePath + "/data",
	}
	go c.init()
	return c
}

func fail(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func (c *client) init() {
	// Create torrent directory
	if _, err := os.Stat(c.torrentsPath); os.IsNotExist(err) {
		err := os.Mkdir(c.torrentsPath, 0755)
		fail(err)
	}
	// Create data directory
	if _, err := os.Stat(c.dataPath); os.IsNotExist(err) {
		err := os.Mkdir(c.dataPath, 0755)
		fail(err)
	}
	// Read all torrent files and load them as stopped torrents
	torrentFiles, err := ioutil.ReadDir(c.torrentsPath)
	fail(err)
	for _, f := range torrentFiles {
		torrentReader, err := os.Open(c.torrentsPath + "/" + f.Name())
		fail(err)
		c.addTorrent(torrentReader)
	}
}

func (c *client) GetTorrents() []TorrentDownload {
	return c.torrents
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
	if r2 == nil {
		fmt.Printf("r2 is nil")
	}
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

func (c *client) AddMagnet(magnetURI string) (TorrentDownload, error) {
	muri, err := parseMagnetURI(magnetURI)
	if err != nil {
		return nil, err
	}
	return NewTorrentFromMagnet(muri), nil
}

func (c *client) AddTorrent(torrentReader io.ReadSeeker) TorrentDownload {
	td, infoHashHex := c.addTorrent(torrentReader)
	c.saveTorrent(torrentReader, infoHashHex)
	c.torrents = append(c.torrents, td)
	return td
}

func (c *client) addTorrent(torrentReader io.ReadSeeker) (TorrentDownload, string) {
	// Parse Torrent
	tor, err := torrent.NewTorrent(torrentReader)
	fail(err)

	// Save Torrent
	td := NewTorrentDownload(tor, c.dataPath)
	infoHashHex := hex.EncodeToString(td.GetInfoHash())
	return td, infoHashHex
}

func (c *client) saveTorrent(torrentReader io.ReadSeeker, infoHashHex string) {
	torrentReader.Seek(0, 0)
	validTorrentData, err := ioutil.ReadAll(torrentReader)
	fail(err)
	file, err := os.OpenFile(c.torrentsPath+"/"+infoHashHex, os.O_CREATE|os.O_RDWR, 0755)
	fail(err)
	_, err = file.Write(validTorrentData)
	fail(err)
}

func (c *client) RemoveTorrent(infoHashHex string) {
	err := os.Remove(c.torrentsPath + "/" + infoHashHex)
	fail(err)
}

func (c *client) RemoveTorrentAndData(infoHashHex string) {
	c.RemoveTorrent(infoHashHex)
	err := os.RemoveAll(c.dataPath + "/" + infoHashHex)
	fail(err)
}

// func (c *client) StopTorrent(torrentID string) {
// 	// c.torrentsStats[torrentID].stopped = true
// 	// c.torrentDownloads[torrentID].StopTorrent()
// }

// func (c *client) StartTorrent(torrentID string) {
// 	// c.torrentsStats[torrentID].stopped = false
// 	c.torrentDownloads[torrentID].Start()
// }

// func (c *client) StopFile(torrentID string, fileIndex int) {
// 	// c.torrentsStats[torrentID].fileStopped[fileIndex] = true
// 	// c.torrentDownloads[torrentID].StopFile()
// }

// func (c *client) StartFile(torrentID string, fileIndex int) {
// 	// c.torrentsStats[torrentID].fileStopped[fileIndex] = false
// 	// c.torrentDownloads[torrentID].StartFile()
// }
