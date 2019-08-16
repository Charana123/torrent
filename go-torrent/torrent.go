package torrent

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"log"
	"math/rand"
	"os"

	bencode "github.com/jackpal/bencode-go"
)

var (
	PEER_ID = make([]byte, 0, 20)
)

func init() {
	PEER_ID = append(PEER_ID, []byte("-GT0001-")...)
	_, err := rand.Read(PEER_ID[8:])
	if err != nil {
		log.Fatalln(err)
	}
}

type Torrent struct {
	metaInfo  *metaInfo
	infoHash  []byte
	numPieces int
}

type metaInfo struct {
	Info struct {
		PieceLength int `bencode:"piece length"`
		Pieces      string
		Private     int
		Name        string
		Length      int
		Md5sum      string
		Files       []struct {
			Length int
			Md5sum string
			Path   []string
		}
	}
	Announce     string
	AnnounceList [][]string `bencode:"announce-list"`
	CreationDate int        `bencode:"creation date"`
	Comment      string
	CreatedBy    string `bencode:"created by"`
	Encoding     string
}

func NewTorrent(filename string) (*Torrent, error) {
	torrent := &Torrent{}

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	metaInfo, err := bencode.Decode(file)
	if err != nil {
		return nil, err
	}
	metaInfoMap, ok := metaInfo.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Malformed torrent file %s", filename)
	}
	infoMap, ok := metaInfoMap["info"]
	if !ok {
		return nil, fmt.Errorf("Malformed torrent file %s", filename)
	}

	infoBencode := &bytes.Buffer{}
	bencode.Marshal(infoBencode, infoMap)

	infoHash := sha1.Sum(infoBencode.Bytes())
	torrent.infoHash = infoHash[:]
	torrent.numPieces = len(torrent.metaInfo.Info.Pieces) / 20

	file.Seek(0, 0)
	err = bencode.Unmarshal(file, &torrent.metaInfo)
	if err != nil {
		return nil, err
	}
	return torrent, nil
}

func (t *Torrent) ServePeers() {

}

// Start/Resume downloading/uploading torrent
func (t *Torrent) Start(serverPeerMChans *serverPeerMChans, port int) chan int {

	// Requests the peer list, spawns another process to send
	// the peer list to the peer manager, manager makes a connection or ignores

	quit := make(chan int)
	progressStats := &progressStats{}
	trackerPeerMChans := &trackerPeerMChans{
		peers: make(chan *peer),
	}
	disk := newDisk(progressStats)
	go disk.start()
	tracker := newTracker(t, progressStats, quit, port, nil, trackerPeerMChans)
	go tracker.start()
	peerMChokeChans := &peerMChokeChans{
		newPeer: make(chan *chokePeerChans),
	}
	peerChokeChans := &peerChokeChans{
		clientChokeStateChan: make(chan *chokeState),
		peerHaveMessagesChan: make(chan *peerHaveMessages),
	}
	peerM := newPeerManager(t, serverPeerMChans, trackerPeerMChans, peerMChokeChans, peerChokeChans)
	go peerM.start()
	choke := newChoke(peerMChokeChans, peerChokeChans)
	go choke.start()
	return quit
}

// Stop downloading/uploading torrent
func (t *Torrent) Stop() {

}

// Delete (potentially only partially) downloaded torrent files
func (t *Torrent) Cleanup() {

}
