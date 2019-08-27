package peer

import (
	"testing"
	"time"
)

func (m *mockPeerManager) GetPeerList() []*PeerInfo {
	args := m.Called()
	return args.Get(0).([]*PeerInfo)
}

func (m *mockWire) SendUnchoke() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockWire) SendChoke() error {
	args := m.Called()
	return args.Error(0)
}

func TestChoke(t *testing.T) {

	pm := &mockPeerManager{}
	w1 := &mockWire{}
	w1.On("SendUnchoke").Return(nil)
	w2 := &mockWire{}
	w3 := &mockWire{}
	w3.On("SendUnchoke").Return(nil)
	w4 := &mockWire{}
	w4.On("SendChoke").Return(nil)

	lastPiece := time.Now().Unix()

	pm.On("GetPeerList").Return([]*PeerInfo{
		&PeerInfo{
			id:   "0.0.0.0",
			wire: w1,
			state: connState{
				peerInterested: true,
				clientChoking:  true,
			},
			lastPiece: lastPiece,
			speed:     10,
		},
		&PeerInfo{
			id:   "0.0.0.1",
			wire: w2,
			state: connState{
				peerInterested: true,
				clientChoking:  false,
			},
			lastPiece: lastPiece,
			speed:     20,
		},
		&PeerInfo{
			id:   "0.0.0.2",
			wire: w3,
			state: connState{
				peerInterested: false,
				clientChoking:  true,
			},
			lastPiece: lastPiece,
			speed:     15,
		},
		&PeerInfo{
			id:   "0.0.0.3",
			wire: w4,
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
	w1.AssertExpectations(t)
	w2.AssertExpectations(t)
	w3.AssertExpectations(t)
	w4.AssertExpectations(t)
}
