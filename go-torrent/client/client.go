package client

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/Charana123/torrent/go-torrent/peer"
	"github.com/Charana123/torrent/go-torrent/server"
	"github.com/Charana123/torrent/go-torrent/torrent"
	"github.com/Charana123/torrent/go-torrent/tracker"
)

type Client interface {
	Start(path string) error
	Stop()
	CleanUp()
}

type client struct {
	quit chan int
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

func NewClient() Client {
	return &client{}
}

// Start/Resume downloading/uploading torrent
func (c *client) Start(path string) error {

	t, err := torrent.NewTorrent(path)
	if err != nil {
		return err
	}

	quit := make(chan int)
	c.quit = quit

	peerMgr := peer.NewPeerManager(t)
	sv, err := server.NewServer(peerMgr, quit)
	go sv.Serve()
	if err != nil {
		return err
	}
	clientIP, err := getExternalIP()
	if err != nil {
		return err
	}
	tr := tracker.NewTracker(t, quit, sv.GetServerPort(), net.ParseIP(clientIP))
	go tr.Start()
	return nil
}

// Stop downloading/uploading torrent
func (t *client) Stop() {

}

// Delete  (potentially only partially) downloaded torrent files
func (t *client) CleanUp() {

}

// TODO - verify which pieces have already been downloaded and verified
