package stats

import (
	"log"
	"sync"

	underscore "github.com/ahl5esoft/golang-underscore"
)

type Stats interface {
	GetTrackerStats() (uploaded int, downloaded int, left int)
	GetPeerStats() (peerStats map[string]*PeerStat)
	UpdatePeer(id string, uploaded int, downloaded int)
}

const (
	PONDERATION_TIME = 10
)

type stats struct {
	sync.Mutex

	trackerStats *TrackerStats
	clientStats  *ClientStats
	peerStats    map[string]*PeerStat
}

type TrackerStats struct {
	TotalUpload   int
	TotalDownload int
	Left          int
}

type ClientStats struct {
	UploadRate       int
	DownloadRate     int
	uploadActivity   [PONDERATION_TIME]int
	downloadActivity [PONDERATION_TIME]int
	i                int
}

type PeerStat struct {
	UploadRate       int
	DownloadRate     int
	currentUpload    int
	currentDownload  int
	uploadActivity   [PONDERATION_TIME]int
	downloadActivity [PONDERATION_TIME]int
	i                int
}

func NewStats(
	uploaded int, downloaded int, left int) Stats {

	return &stats{
		trackerStats: &TrackerStats{
			TotalUpload:   uploaded,
			TotalDownload: downloaded,
			Left:          left,
		},
		clientStats: &ClientStats{},
		peerStats:   make(map[string]*PeerStat),
	}
}

func (s *stats) GetTrackerStats() (int, int, int) {
	return s.trackerStats.TotalUpload, s.trackerStats.TotalDownload, s.trackerStats.Left
}

func (s *stats) UpdatePeer(id string, uploaded int, downloaded int) {
	s.Lock()
	defer s.Unlock()

	peerStat, ok := s.peerStats[id]
	if !ok {
		peerStat = &PeerStat{}
		s.peerStats[id] = peerStat
	}
	peerStat.currentUpload += uploaded
	peerStat.currentDownload += downloaded
}

func (s *stats) RemovePeer(id string) {
	s.Lock()
	defer s.Unlock()

	delete(s.peerStats, id)
}

func sumReduce(acc int, x, _ int) int {
	return acc + x
}

func (s *stats) GetPeerStats() map[string]*PeerStat {
	s.Lock()
	defer s.Unlock()

	clientCurrentUpload := 0
	clientCurrentDownload := 0
	for _, peerStat := range s.peerStats {
		peerStat.uploadActivity[peerStat.i] = peerStat.currentUpload
		peerStat.downloadActivity[peerStat.i] = peerStat.currentDownload
		underscore.Chain(peerStat.uploadActivity).Reduce(0, sumReduce).Value(&peerStat.UploadRate)
		peerStat.UploadRate /= PONDERATION_TIME
		underscore.Chain(peerStat.downloadActivity).Reduce(0, sumReduce).Value(&peerStat.DownloadRate)
		peerStat.DownloadRate /= PONDERATION_TIME
		peerStat.i = (peerStat.i + 1) % PONDERATION_TIME

		clientCurrentDownload += peerStat.currentUpload
		clientCurrentUpload += peerStat.currentDownload
		peerStat.currentUpload = 0
		peerStat.currentDownload = 0
	}

	s.clientStats.uploadActivity[s.clientStats.i] = clientCurrentUpload
	s.clientStats.downloadActivity[s.clientStats.i] = clientCurrentDownload
	underscore.Chain(s.clientStats.uploadActivity).Reduce(0, sumReduce).Value(&s.clientStats.UploadRate)
	s.clientStats.UploadRate /= PONDERATION_TIME
	underscore.Chain(s.clientStats.downloadActivity).Reduce(0, sumReduce).Value(&s.clientStats.DownloadRate)
	s.clientStats.DownloadRate /= PONDERATION_TIME
	s.clientStats.i = (s.clientStats.i + 1) % PONDERATION_TIME

	s.trackerStats.TotalUpload += clientCurrentUpload
	s.trackerStats.TotalDownload += clientCurrentDownload
	log.Printf("Download: %d KBps, Upload: %d KBps\n", s.clientStats.DownloadRate/2014, s.clientStats.UploadRate/2014)
	return s.peerStats
}
