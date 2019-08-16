package torrent

type disk struct {
	progressStats *progressStats
	// peerChans *diskPeerChans
}

func newDisk(progressStats *progressStats) *disk {
	return &disk{
		progressStats: progressStats,
	}
}

func (d *disk) start() {

}
