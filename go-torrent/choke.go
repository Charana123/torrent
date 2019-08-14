package torrent

type chokePeerChans struct {
	// e.g. request piece from peer
}

type choke struct {
	peerMChans *peerMChokeChans
}

func newChoke(peerMChans *peerMChokeChans) *choke {
	return &choke{
		peerMChans: peerMChans,
	}
}
