package torrent

type choke struct {
	peerMChans *peerMChokeChans
	peerChans  *peerChokeChans
}

func newChoke(peerMChans *peerMChokeChans, peerChans *peerChokeChans) *choke {
	return &choke{
		peerMChans: peerMChans,
		peerChans:  peerChans,
	}
}

func (c *choke) start() {
	for {
		select {
		// chokeState
		}
	}
}
