package torrent

import (
	"log"
	"net"
	"time"
)

type serverPeerMChans struct {
	peers chan *peer
}

type Server struct {
	port      int
	listener  *net.TCPListener
	peerMChan *serverPeerMChans
	quit      chan int
}

// give as input, the connection that sends peer connections to torrent
func NewServer(quit chan int) (*Server, *serverPeerMChans, int, error) {
	sv := &Server{
		quit: quit,
		peerMChan: &serverPeerMChans{
			peers: make(chan *peer),
		},
	}
	var err error
	sv.listener, err = net.ListenTCP("tcp4", &net.TCPAddr{})
	if err != nil {
		return nil, nil, 0, err
	}
	sv.port = sv.listener.Addr().(*net.TCPAddr).Port
	return sv, sv.peerMChan, sv.port, nil
}

func (sv *Server) Serve() {
	go func() {
		for {
			select {
			case <-sv.quit:
				log.Println("Safely terminating peer listener")
				return
			default:
				sv.listener.SetDeadline(time.Now().Add(time.Second))
				conn, err := sv.listener.AcceptTCP()
				if err != nil {
					log.Panicln("Error! Terminating peer listener, please restart")
					return
				}
				peer := &peer{}
				peer.conn = conn
				peer.id = conn.RemoteAddr().String()
				// Non-blocking
				go func() { sv.peerMChan.peers <- peer }()
			}
		}
	}()
}
