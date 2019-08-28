package piece

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/boljen/go-bitmap"

	"github.com/Charana123/torrent/go-torrent/storage"
	"github.com/Charana123/torrent/go-torrent/wire"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Charana123/torrent/go-torrent/torrent"
)

type mockDisk struct {
	storage.Storage
	mock.Mock
}

func (m *mockDisk) WritePieceRequest(pieceIndex int, data []byte) error {
	args := m.Called(pieceIndex, data)
	return args.Error(0)
}

type mockWire struct {
	wire.Wire
	mock.Mock
}

func (m *mockWire) SendRequest(pieceIndex, begin, length int) error {
	args := m.Called(pieceIndex, begin, length)
	return args.Error(0)
}

func (m *mockWire) SendUnInterested() error {
	args := m.Called()
	return args.Error(0)
}

func TestPieceCompleted(t *testing.T) {
	tor := &torrent.Torrent{
		NumPieces: 3,
		MetaInfo: torrent.MetaInfo{
			Info: torrent.Info{
				PieceLength: 65536, // 2^16
			},
		},
	}
	block1 := make([]byte, BLOCK_SIZE)
	block2 := make([]byte, BLOCK_SIZE)
	block3 := make([]byte, BLOCK_SIZE)
	block4 := make([]byte, BLOCK_SIZE)
	for i := 0; i < BLOCK_SIZE; i++ {
		block1[i] = 1
		block2[i] = 2
		block3[i] = 3
		block4[i] = 4
	}

	disk := &mockDisk{}
	disk.On("WritePieceRequest", 1, mock.MatchedBy(func(piece []byte) bool {
		if len(piece) != BLOCK_SIZE*4 {
			return false
		}
		if !bytes.Equal(piece[:BLOCK_SIZE], block1) ||
			!bytes.Equal(piece[BLOCK_SIZE:BLOCK_SIZE*2], block2) ||
			!bytes.Equal(piece[BLOCK_SIZE*2:BLOCK_SIZE*3], block3) ||
			!bytes.Equal(piece[BLOCK_SIZE*3:], block4) {
			return false
		}
		return true
	})).Return(nil).Once()

	pm := NewRarestFirstPieceManager(tor, disk)

	MAX_OUTSTANDING_REQUESTS = 3
	wire := &mockWire{}
	wire.On("SendRequest", 1, 0, BLOCK_SIZE).Return(nil).Once()
	wire.On("SendRequest", 1, 1, BLOCK_SIZE).Return(nil).Once()
	wire.On("SendRequest", 1, 2, BLOCK_SIZE).Return(nil).Once()
	wire.On("SendRequest", 1, 3, BLOCK_SIZE).Return(nil).Once()
	wire.On("SendUnInterested").Return(nil).Once()
	peerID := "0.0.0.0"
	peerBitField := bitmap.New(3)
	peerBitField.Set(1, true)

	// peer unchokes client
	pm.SendBlockRequests(peerID, wire, &peerBitField)

	// peer sends block message, client responds by sending another piece (x 4)
	pm.WriteBlock(peerID, 1, 1, block2)
	pm.SendBlockRequests(peerID, wire, &peerBitField)
	pm.WriteBlock(peerID, 1, 0, block1)
	pm.SendBlockRequests(peerID, wire, &peerBitField)
	pm.WriteBlock(peerID, 1, 2, block3)
	pm.SendBlockRequests(peerID, wire, &peerBitField)
	pm.WriteBlock(peerID, 1, 3, block4)
	pm.SendBlockRequests(peerID, wire, &peerBitField)

	disk.AssertExpectations(t)
	wire.AssertExpectations(t)
}

func TestPeerChoked(t *testing.T) {

	tor := &torrent.Torrent{
		NumPieces: 3,
		MetaInfo: torrent.MetaInfo{
			Info: torrent.Info{
				PieceLength: 65536, // 2^16
			},
		},
	}
	pieces := make([]byte, tor.NumPieces*20)
	tor.MetaInfo.Info.Pieces = string(pieces)

	block1 := make([]byte, BLOCK_SIZE)
	block2 := make([]byte, BLOCK_SIZE)
	block3 := make([]byte, BLOCK_SIZE)
	block4 := make([]byte, BLOCK_SIZE)
	for i := 0; i < BLOCK_SIZE; i++ {
		block1[i] = 1
		block2[i] = 2
		block3[i] = 3
		block4[i] = 4
	}

	pm := NewRarestFirstPieceManager(tor, nil)

	MAX_OUTSTANDING_REQUESTS = 2
	wire1 := &mockWire{}
	wire1.On("SendRequest", 1, 0, BLOCK_SIZE).Return(nil).Once()
	wire1.On("SendRequest", 1, 1, BLOCK_SIZE).Return(nil).Once()
	wire1.On("SendRequest", 1, 2, BLOCK_SIZE).Return(nil).Once()
	wire2 := &mockWire{}
	wire2.On("SendRequest", 1, 0, BLOCK_SIZE).Return(nil).Once()
	wire2.On("SendRequest", 1, 2, BLOCK_SIZE).Return(nil).Once()
	wire2.On("SendRequest", 1, 3, BLOCK_SIZE).Return(nil).Once()
	peerID1 := "0.0.0.0"
	peerID2 := "0.0.0.1"

	peerBitField1 := bitmap.New(3)
	peerBitField1.Set(1, true)
	peerBitField2 := bitmap.New(3)
	peerBitField2.Set(1, true)

	// peer1 unchokes client
	pm.SendBlockRequests(peerID1, wire1, &peerBitField1)
	// peer1 sends block message, client responds by sending another piece
	pm.WriteBlock(peerID1, 1, 1, block2)
	pm.SendBlockRequests(peerID1, wire1, &peerBitField1)
	// peer1 chokes client
	pm.PeerChoked(peerID1)
	// peer2 unchokes client
	pm.SendBlockRequests(peerID2, wire2, &peerBitField2)
	// peer2 sends block message, client responds by sending another piece
	pm.WriteBlock(peerID2, 1, 0, block3)
	pm.SendBlockRequests(peerID2, wire2, &peerBitField2)
	pm.WriteBlock(peerID2, 1, 2, block4)
	pm.SendBlockRequests(peerID2, wire2, &peerBitField2)
	err := pm.WriteBlock(peerID2, 1, 3, block4)
	assert.Equal(t, err, fmt.Errorf("checksum failed"))
	pm.SendBlockRequests(peerID2, wire2, &peerBitField2)

	wire1.AssertExpectations(t)
	wire2.AssertExpectations(t)
}
