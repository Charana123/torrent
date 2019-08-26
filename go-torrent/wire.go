package torrent

import (
	"bytes"
	"encoding/binary"
	"net"
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
	SendChoke()
	SendUnchoke()
	SendInterested()
	SendUnInterested()
	SendRequest(pieceIndex, blockIndex, length int)
	SendBitField(bitfield []byte)
	SendBlock(pieceIndex, begin int, block []byte)
}

type wire struct {
	conn net.Conn
	peer Peer
}

func (w *wire) SendChoke() {
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, int32(1))
	binary.Write(b, binary.BigEndian, uint8(CHOKE))
	w.sendMessage(b.Bytes())
}

func (w *wire) SendUnchoke() {
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, int32(1))
	binary.Write(b, binary.BigEndian, uint8(UNCHOKE))
	w.sendMessage(b.Bytes())
}

func (w *wire) SendInterested() {
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, int32(1))
	binary.Write(b, binary.BigEndian, uint8(INTERESTED))
	w.sendMessage(b.Bytes())
}

func (w *wire) SendUnInterested() {
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, int32(1))
	binary.Write(b, binary.BigEndian, uint8(NOT_INTERESTED))
	w.sendMessage(b.Bytes())
}

func (w *wire) SendBlock(pieceIndex, begin int, block []byte) {
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, int32(9+len(block)))
	binary.Write(b, binary.BigEndian, uint8(BLOCK))
	binary.Write(b, binary.BigEndian, block)
	w.sendMessage(b.Bytes())
}

func (w *wire) SendBitField(bitfield []byte) {
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, int32(1+len(bitfield)))
	binary.Write(b, binary.BigEndian, uint8(BITFIELD))
	binary.Write(b, binary.BigEndian, bitfield)
	w.sendMessage(b.Bytes())
}

func (w *wire) SendRequest(pieceIndex, blockIndex, length int) {
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, int32(13))
	binary.Write(b, binary.BigEndian, uint8(REQUEST))
	binary.Write(b, binary.BigEndian, int32(pieceIndex))
	binary.Write(b, binary.BigEndian, int32(blockIndex))
	binary.Write(b, binary.BigEndian, int32(length))
	w.sendMessage(b.Bytes())
}

func (w *wire) sendMessage(msg []byte) {
	_, err := w.conn.Write(msg)
	if err != nil {
		w.peer.Stop()
	}
}
