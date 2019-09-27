package server

import (
	"fmt"
	"log"
	"net"

	"github.com/Charana123/torrent/go-torrent/peer"
)

type Server interface {
	Serve()
	GetServerPort() int
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

func NewServer(
	pm peer.PeerManager,
	quit chan int) (Server, error) {

	sv := &server{
		pm:   pm,
		quit: quit,
	}
	listener, err := listen("tcp4", "")
	sv.listener = listener
	if err != nil {
		return nil, err
	}
	sv.port = sv.listener.Addr().(*net.TCPAddr).Port
	fmt.Println("sv.port", sv.port)
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
				tcpAddr := conn.RemoteAddr().(*net.TCPAddr)
				sv.pm.AddPeer(tcpAddr.IP.String(), conn)
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

func (sv *server) GetServerPort() int {
	return sv.port
}
