package peer

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/Charana123/torrent/go-torrent/piece"
	"github.com/Charana123/torrent/go-torrent/wire"

	"github.com/boljen/go-bitmap"

	"github.com/Charana123/torrent/go-torrent/torrent"
	"github.com/stretchr/testify/mock"
)

type mockWire struct {
	wire.Wire
	mock.Mock
}

func (m *mockWire) SendHandshake(length uint8, protocol string, infohash []byte, peerID []byte) error {
	args := m.Called(length, protocol, infohash, peerID)
	return args.Error(0)
}

func (m *mockWire) ReadHandshake() (uint8, string, []byte, []byte, error) {
	args := m.Called()
	return args.Get(0).(uint8), args.String(1), args.Get(2).([]byte), args.Get(3).([]byte), args.Error(4)
}

func (m *mockWire) SendBitField(bitfield []byte) error {
	args := m.Called(bitfield)
	return args.Error(0)
}

func (m *mockWire) ReadMessage() (int, byte, []byte, error) {
	args := m.Called()
	return args.Int(0), args.Get(1).(byte), args.Get(2).([]byte), args.Error(3)
}

func (m *mockWire) Close() {
	m.Called()
}

type mockPieceManager struct {
	piece.PieceManager
	mock.Mock
}

func (m *mockPieceManager) GetBitField() []byte {
	args := m.Called()
	return args.Get(0).([]byte)
}

func (m *mockPieceManager) PeerStopped(id string, peerBitfield *bitmap.Bitmap) {
	m.Called(id, peerBitfield)
}

// type mockDisk struct {
// 	Disk
// }

func preFunc(t *testing.T) (*torrent.Torrent, *mockPieceManager, *mockWire) {
	mockWire := &mockWire{}
	newWire = func(conn net.Conn) wire.Wire {
		return mockWire
	}

	tor := &torrent.Torrent{
		NumPieces: 10,
		InfoHash:  make([]byte, 20),
	}

	mockWire.On("SendHandshake",
		uint8(19),
		"BitTorrent protocol",
		tor.InfoHash,
		torrent.PEER_ID).Return(nil)

	mockWire.On("ReadHandshake").Return(
		uint8(19),
		"BitTorrent protocol",
		tor.InfoHash,
		([]uint8)(nil),
		nil)

	mockPieceMgr := &mockPieceManager{}
	bitfield := bitmap.New(tor.NumPieces).Data(false)
	mockPieceMgr.On("GetBitField").Return(bitfield)

	mockWire.On("SendBitField", bitfield).Return(nil)

	return tor, mockPieceMgr, mockWire
}

type mockPeerManager struct {
	PeerManager
	mock.Mock
}

func (m *mockPeerManager) RemovePeer(id string) {
	m.Called(id)
}

func TestPeerDisconnect(t *testing.T) {
	tor, mockPieceMgr, mockWire := preFunc(t)

	connSIG := make(chan time.Time)
	mockWire.On("ReadMessage").WaitUntil(connSIG).Return(0, byte(0), ([]uint8)(nil), errors.New(""))

	peerID := "0.0.0.0"
	mockPeerMgr := &mockPeerManager{}
	mockPeerMgr.On("RemovePeer", peerID).Return()

	mockPieceMgr.On("PeerStopped", peerID, (*bitmap.Bitmap)(nil)).Return()

	mockWire.On("Close").Return()
	NewPeer(
		peerID,
		mockWire,
		tor,
		nil,
		mockPeerMgr,
		mockPieceMgr,
	)
	<-time.After(time.Second)
	close(connSIG)
	<-time.After(time.Second)
	mockWire.AssertExpectations(t)
	mockPieceMgr.AssertExpectations(t)
	mockPeerMgr.AssertExpectations(t)
}
