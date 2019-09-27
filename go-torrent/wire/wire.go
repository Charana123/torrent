package wire

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"time"
)

const (
	CHOKE          = 0
	UNCHOKE        = 1
	INTERESTED     = 2
	NOT_INTERESTED = 3
	HAVE           = 4
	BITFIELD       = 5
	REQUEST        = 6
	BLOCK          = 7
	CANCEL         = 8
	PORT           = 9
)

type Wire interface {
	// Reading
	ReadHandshake() (uint8, string, []byte, []byte, error)
	ReadMessage() (int32, byte, []byte, error)

	// Writing
	SendHandshake(length uint8, protocol string, infohash []byte, peerID []byte) error
	SendKeepAlive() error
	SendChoke() error
	SendUnchoke() error
	SendInterested() error
	SendUnInterested() error
	SendHave(pieceIndex int) error
	SendBitField(bitfield []byte) error
	SendRequest(pieceIndex, begin, length int) error
	SendBlock(pieceIndex, begin int, block []byte) error
	// SendCancel(pieceIndex, begin, length int) error

	// Other
	GetLastMessageSent() (lastMessageSent time.Time)
	Close()
}

type wire struct {
	conn            *net.TCPConn
	timeoutDuration time.Duration
	lastMessageSent time.Time
}

func NewWire(
	conn *net.TCPConn,
	timeoutDuration time.Duration) Wire {

	return &wire{
		conn:            conn,
		timeoutDuration: timeoutDuration,
	}
}

// 1 + 19 + 8 + 20 + 20
type Handshake struct {
	Len      uint8
	Protocol [19]byte
	Reserved [8]uint8
	InfoHash [20]byte
	PeerID   [20]byte
}

func (w *wire) GetLastMessageSent() time.Time {
	return w.lastMessageSent
}

func (w *wire) SendKeepAlive() error {
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, int32(1))
	return w.sendMessage(b.Bytes())
}

func (w *wire) SendHave(pieceIndex int) error {
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, int32(5))
	binary.Write(b, binary.BigEndian, uint8(4))
	binary.Write(b, binary.BigEndian, int32(pieceIndex))
	return w.sendMessage(b.Bytes())
}

func (w *wire) SendHandshake(length uint8, protocol string, infohash []byte, peerID []byte) error {
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, length)
	binary.Write(b, binary.BigEndian, []byte(protocol))
	binary.Write(b, binary.BigEndian, make([]byte, 8))
	binary.Write(b, binary.BigEndian, infohash)
	binary.Write(b, binary.BigEndian, peerID)
	return w.sendMessage(b.Bytes())
}

func (w *wire) Close() {
	w.conn.Close()
}

func (w *wire) ReadHandshake() (uint8, string, []byte, []byte, error) {
	h := &Handshake{}
	w.conn.SetReadDeadline(time.Now().Add(w.timeoutDuration))
	data := make([]byte, 68)
	_, err := io.ReadFull(w.conn, data)
	if err != nil {
		return 0, "", nil, nil, err
	}
	err = binary.Read(bytes.NewBuffer(data), binary.BigEndian, h)
	return h.Len, string(h.Protocol[:]), h.InfoHash[:], h.PeerID[:], nil
}

func (w *wire) ReadMessage() (int32, byte, []byte, error) {
	w.conn.SetReadDeadline(time.Now().Add(w.timeoutDuration))

	var length int32
	err1 := binary.Read(w.conn, binary.BigEndian, &length)
	if length == 0 {
		return length, 0, nil, nil
	}
	if err1 != nil {
		return 0, 0, nil, err1
	}
	var ID uint8
	err2 := binary.Read(w.conn, binary.BigEndian, &ID)
	if err2 != nil {
		return 0, 0, nil, err2
	}

	payload := make([]byte, length-1)
	_, err3 := io.ReadFull(w.conn, payload)
	if err3 != nil {
		return 0, 0, nil, err3
	}
	return length, ID, payload, nil
}

func (w *wire) SendChoke() error {
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, int32(1))
	binary.Write(b, binary.BigEndian, uint8(CHOKE))
	return w.sendMessage(b.Bytes())
}

func (w *wire) SendUnchoke() error {
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, int32(1))
	binary.Write(b, binary.BigEndian, uint8(UNCHOKE))
	return w.sendMessage(b.Bytes())
}

func (w *wire) SendInterested() error {
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, int32(1))
	binary.Write(b, binary.BigEndian, uint8(INTERESTED))
	return w.sendMessage(b.Bytes())
}

func (w *wire) SendUnInterested() error {
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, int32(1))
	binary.Write(b, binary.BigEndian, uint8(NOT_INTERESTED))
	return w.sendMessage(b.Bytes())
}

func (w *wire) SendBlock(pieceIndex, begin int, block []byte) error {
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, int32(9+len(block)))
	binary.Write(b, binary.BigEndian, uint8(BLOCK))
	binary.Write(b, binary.BigEndian, int32(pieceIndex))
	binary.Write(b, binary.BigEndian, int32(begin))
	binary.Write(b, binary.BigEndian, block)
	return w.sendMessage(b.Bytes())
}

func (w *wire) SendBitField(bitfield []byte) error {
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, int32(1+len(bitfield)))
	binary.Write(b, binary.BigEndian, uint8(BITFIELD))
	binary.Write(b, binary.BigEndian, bitfield)
	return w.sendMessage(b.Bytes())
}

func (w *wire) SendRequest(pieceIndex, begin, length int) error {
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, int32(13))
	binary.Write(b, binary.BigEndian, uint8(REQUEST))
	binary.Write(b, binary.BigEndian, int32(pieceIndex))
	binary.Write(b, binary.BigEndian, int32(begin))
	binary.Write(b, binary.BigEndian, int32(length))
	return w.sendMessage(b.Bytes())
}

func (w *wire) sendMessage(msg []byte) error {
	w.lastMessageSent = time.Now()
	w.conn.SetWriteDeadline(time.Now().Add(w.timeoutDuration))
	_, err := w.conn.Write(msg)
	if err != nil {
		return err
	}
	return nil
}
