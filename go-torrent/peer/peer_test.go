package peer

import (
	"net"
	"testing"

	"github.com/stretchr/testify/mock"
)

type mockConn struct {
	net.Conn
	mock.Mock
}

func (m *mockConn) Write(p []byte) (n int, err error) {
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}

func (m *mockConn) Read(p []byte) (n int, err error) {
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}

// type mockDisk struct {
// 	Disk
// }

func TestNewPeer(t *testing.T) {
	// mockConn := &mockConn{}
	// mockConn.On("Write").Return(68, nil)

	// torrent := &Torrent{
	// 	numPieces: 10,
	// 	infoHash:  make([]byte, 20),
	// }

	// hreq := &handshake{}
	// hreq.Len = 19
	// copy(hreq.Protocol[:], "BitTorrent protocol")
	// copy(hreq.InfoHash[:], p.torrent.infoHash)
	// copy(hreq.PeerID[:], [20]byte{})
	// binary.Write

	// mockConn.On("Read").Return(68, nil)

	// dial = func(network, address string) (net.Conn, err) {

	// }

	// disk := &mockDisk{}

	// newPeer(
	// 	"0.0.0.0",
	// 	nil,
	// 	disk,
	// 	torrent,
	// )
}
