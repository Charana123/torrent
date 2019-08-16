package torrent

type disk struct {
	progressStats *progressStats
	peerChans     *peerDiskChans
}

func newDisk(progressStats *progressStats, peerChans *peerDiskChans) *disk {
	return &disk{
		progressStats: progressStats,
		peerChans:     peerChans,
	}
}

func (d *disk) start() {

}

// How this module works
// Core - Responds to peer block read and piece write requests,
// also the quit signal handler
// Can also maintains the download and upload information
// (how much has been written and read), a structure that the tracker
// has access to and is read only.
