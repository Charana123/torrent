package torrent

import "net"

type Peer struct {
	Addr               *net.TCPAddr
	PeerHasChoked      bool
	PerIsInterested    bool
	ClientIsInterested bool
	ClientHasChoked    bool
}

// handeshake and start handling messages from each peer connection
// send bitfield message
// from each thread update the global list of peer that have unchoked
// Use a manager thread to organise the downloading and uploading
