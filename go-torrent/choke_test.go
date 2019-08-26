package torrent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
)

type mockPeerManager struct {
	PeerManager
	mock.Mock
}

func (m *mockPeerManager) GetPeerList() []*PeerInfo {
	args := m.Called()
	return args.Get(0).([]*PeerInfo)
}

type mockPeer struct {
	Peer
	mock.Mock
}

func (m *mockPeer) SendUnchoke() {
	m.Called()
}

func (m *mockPeer) SendChoke() {
	m.Called()
}

func TestChoke(t *testing.T) {

	pm := &mockPeerManager{}
	p1 := &mockPeer{}
	p1.On("SendUnchoke").Return()
	p2 := &mockPeer{}
	p3 := &mockPeer{}
	p3.On("SendUnchoke").Return()
	p4 := &mockPeer{}
	p4.On("SendChoke").Return()

	lastPiece := time.Now().Unix()

	pm.On("GetPeerList").Return([]*PeerInfo{
		&PeerInfo{
			id:   "0.0.0.0",
			peer: p1,
			state: connState{
				peerInterested: true,
				clientChoking:  true,
			},
			lastPiece: lastPiece,
			speed:     10,
		},
		&PeerInfo{
			id:   "0.0.0.1",
			peer: p2,
			state: connState{
				peerInterested: true,
				clientChoking:  false,
			},
			lastPiece: lastPiece,
			speed:     20,
		},
		&PeerInfo{
			id:   "0.0.0.2",
			peer: p3,
			state: connState{
				peerInterested: false,
				clientChoking:  true,
			},
			lastPiece: lastPiece,
			speed:     15,
		},
		&PeerInfo{
			id:   "0.0.0.3",
			peer: p4,
			state: connState{
				peerInterested: false,
				clientChoking:  false,
			},
			lastPiece: lastPiece,
			speed:     5,
		},
	})

	quit := make(chan int)
	newChoke(pm, quit)
	<-time.After(time.Duration(2) * time.Second)
	close(quit)
	pm.AssertExpectations(t)
	p1.AssertExpectations(t)
	p2.AssertExpectations(t)
	p3.AssertExpectations(t)
	p4.AssertExpectations(t)
}
