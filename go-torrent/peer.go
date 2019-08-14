package torrent

import "net"

type peerChokeChans struct {
	// e.g. notify choke algorithm of current peer state
}

type peer struct {
	id             string
	conn           net.Conn
	toChokeChans   *peerChokeChans
	fromChokeChans *chokePeerChans
}

func (p *peer) start() {
	// send handshake to peer
	// obtain bitfield from choke algorithm
	// send bitfield to peer
	// spawn thread to process incoming messages
	// use this thread to process other events
}
