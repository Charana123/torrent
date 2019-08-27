package server

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
)

type mockListener struct {
	net.Listener
	mock.Mock
}

func (m *mockListener) Accept() (net.Conn, error) {
	args := m.Called()
	return args.Get(0).(net.Conn), args.Error(1)
}

func (m *mockListener) Addr() net.Addr {
	args := m.Called()
	return args.Get(0).(net.Addr)
}

func (m *mockListener) Close() error {
	args := m.Called()
	return args.Error(0)
}

type mockPM struct {
	PeerManager
	mock.Mock
}

func (pm *mockPM) AddPeer(peer *peer) {
	pm.Called(peer)
}

type mockNetError struct {
	net.Error
}

func (pm *mockPM) Timeout() bool {
	return true
}

type mockConn struct {
	net.Conn
}

func (pm *mockConn) RemoteAddr() net.Addr {
	return &mockAddr{}
}

type mockAddr struct {
	net.Addr
}

func (pm *mockAddr) String() string {
	return "0.0.0.0"
}

func TestServer(t *testing.T) {
	ml := &mockListener{}
	ml.On("Addr").Return(&net.TCPAddr{Port: 8181}, nil)
	ml.On("Accept").Return(&mockConn{}, nil)
	ml.On("Accept").After(time.Second * time.Duration(3)).Return(&mockNetError{})
	ml.On("Close").Return(nil)

	listen = func(network, address string) (net.Listener, error) {
		return ml, nil
	}
	pm := &mockPM{}
	pm.On("AddPeer", mock.MatchedBy(func(peer *peer) bool {
		return peer.id == "0.0.0.0"
	})).Return()

	quit := make(chan int)
	newServer(pm, quit)
	<-time.After(time.Second)
	close(quit)
	<-time.After(time.Second)
	ml.AssertExpectations(t)
	pm.AssertExpectations(t)
}
