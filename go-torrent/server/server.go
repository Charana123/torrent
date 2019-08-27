package server

import (
	"log"
	"net"

	"github.com/Charana123/torrent/go-torrent/peer"
)

type Server interface {
	GetServerPort() int
	Serve()
}

func (sv *server) GetServerPort() int {
	return sv.port
}

type server struct {
	port     int
	listener net.Listener
	quit     chan int
	pm       peer.PeerManager
}

var (
	listen = net.Listen
)

// give as input, the connection that sends peer connections to torrent
func NewServer(
	pm peer.PeerManager,
	quit chan int) (Server, error) {

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
	return sv, nil
}

func (sv *server) Serve() {
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
				sv.pm.AddPeer(conn.RemoteAddr().String(), conn)
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
