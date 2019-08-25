package torrent

import (
	"log"
	"net"
)

type server struct {
	port     int
	listener net.Listener
	quit     chan int
	pm       PeerManager
}

var (
	listen = net.Listen
)

// give as input, the connection that sends peer connections to torrent
func newServer(pm PeerManager, quit chan int) (*server, error) {
	sv := &server{
		pm:   pm,
		quit: quit,
	}
	var err error
	sv.listener, err = listen("tcp4", "")
	if err != nil {
		return nil, err
	}
	sv.port = sv.listener.Addr().(*net.TCPAddr).Port
	sv.serve()
	return sv, nil
}

func (sv *server) serve() {
	go func() {
		sig := make(chan int)
		for {
			go func() {
				conn, err := sv.listener.Accept()
				if err != nil {
					// If connection was closed, stop goroutine
					if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
						return
					}
					sig <- 1
					return
				}
				peer := &peer{}
				peer.conn = conn
				peer.id = conn.RemoteAddr().String()
				sv.pm.AddPeer(peer)
				sig <- 0
			}()

			select {
			case <-sv.quit:
				sv.listener.Close()
				log.Println("Safely terminating peer listener")
				return
			case c := <-sig:
				if c == 1 {
					log.Panicln("Error! Terminating peer listener, please restart")
					return
				}
				continue
			}
		}
	}()
}
