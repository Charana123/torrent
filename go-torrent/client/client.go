package client

import (
	"encoding/hex"
	"strings"

	"github.com/Charana123/torrent/go-torrent/download"
	"github.com/Charana123/torrent/go-torrent/torrent"
)

type Client interface {
	AddTorrent(torrentStr string)
}

type torrentStats struct {
	stopped bool
	fileStopped []bool
}

type client struct {
	downloads     map[string]download.Download
	torrentsStats map[string]torrentStats
}

func NewClient() {
	return &client{}
}

func (c *client) AddTorrent(torrentBuff bytes.Buffer) error {
	tor, err := torrent.NewTorrent(torrentBuff)
	if err != nil {
		return err
	}
	infoHashHex := hex.EncodeToString(tor.InfoHash)
	c.downloads[infoHashHex] = download.NewDownload(tor)
	c.torrents[infoHashHex] = &torrentStats{
		fileStopped: make([]bool, len(tor.MetaInfo.Info.Files))
	}
	return nil
}

func (c *client) StopTorrent(torrentID string) {
	c.torrentsStats[torrentID].stopped = true
	c.downloads[torrentID].StopTorrent()
}

func (c *client) StartTorrent(torrentID string) {
	c.torrentsStats[torrentID].stopped = false
	c.downloads[torrentID].StartTorrent()
}

func (c *client) StopFile(torrentID string, fileIndex int){
	c.torrentsStats[torrentID].fileStopped[fileIndex] = true
	c.downloads[torrentID].StopFile()
}

func (c *client) StartFile(torrentID string, fileIndex int){
	c.torrentsStats[torrentID].fileStopped[fileIndex] = false
	c.downloads[torrentID].StartFile()
}