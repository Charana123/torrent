package client

import (
	"encoding/hex"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/Charana123/torrent/go-torrent/torrent"
)

type Client interface {
	// AddTorrent(torrentt torrent.Torrent)
	// RemoveTorrent(infoHashHex string)
	// RemoveTorrentAndData(infoHashHex string)
	// StartTorrent(torrentID string)
	// StopTorrent(torrentID string)
	// StopFile(torrentID string, fileIndex int)
	// StartFile(torrentID string, fileIndex int)
}

type client struct {
	torrentDownloads map[string]TorrentDownload
	torrentsPath     string
	dataPath         string
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

// Add Torrent File
func (c *client) AddTorrent(torrentReader io.ReadSeeker) {
	infoHashHex := c.addTorrent(torrentReader)
	c.saveTorrent(torrentReader, infoHashHex)
}

func (c *client) addTorrent(torrentReader io.ReadSeeker) string {
	// Parse Torrent
	tor, err := torrent.NewTorrent(torrentReader)
	fail(err)

	// Save Torrent
	infoHashHex := hex.EncodeToString(tor.InfoHash)
	c.torrentDownloads[infoHashHex] = NewTorrentDownload(tor)
	return infoHashHex
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

// Remove Torrent File
func (c *client) RemoveTorrent(infoHashHex string) {
	err := os.Remove(c.torrentsPath + "/" + infoHashHex)
	fail(err)
}

// Remove Torrent File and Torrent Data
func (c *client) RemoveTorrentAndData(infoHashHex string) {
	c.RemoveTorrent(infoHashHex)
	err := os.Remove(c.dataPath + "/" + infoHashHex)
	fail(err)
}

func (c *client) VerifyData(infoHashHex string) {

}

func (c *client) StopTorrent(torrentID string) {
	// c.torrentsStats[torrentID].stopped = true
	// c.torrentDownloads[torrentID].StopTorrent()
}

func (c *client) StartTorrent(torrentID string) {
	// c.torrentsStats[torrentID].stopped = false
	// c.torrentDownloads[torrentID].StartTorrent()
}

func (c *client) StopFile(torrentID string, fileIndex int) {
	// c.torrentsStats[torrentID].fileStopped[fileIndex] = true
	// c.torrentDownloads[torrentID].StopFile()
}

func (c *client) StartFile(torrentID string, fileIndex int) {
	// c.torrentsStats[torrentID].fileStopped[fileIndex] = false
	// c.torrentDownloads[torrentID].StartFile()
}
