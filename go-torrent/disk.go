package torrent

type disk struct {
	stats torrentStats
}

type torrentStats struct {
	uploaded   int
	downloaded int
	left       int
}

func newDisk() *disk {
	return &disk{}
}
