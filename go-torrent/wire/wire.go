package wire

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
	SendChoke() error
	SendUnchoke() error
	SendInterested() error
	SendUnInterested() error
	SendRequest(pieceIndex, blockIndex, length int) error
	SendBitField(bitfield []byte) error
	SendBlock(pieceIndex, begin int, block []byte) error
}

type wire struct {
	conn net.Conn
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

func (w *wire) SendRequest(pieceIndex, blockIndex, length int) error {
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, int32(13))
	binary.Write(b, binary.BigEndian, uint8(REQUEST))
	binary.Write(b, binary.BigEndian, int32(pieceIndex))
	binary.Write(b, binary.BigEndian, int32(blockIndex))
	binary.Write(b, binary.BigEndian, int32(length))
	return w.sendMessage(b.Bytes())
}

func (w *wire) sendMessage(msg []byte) error {
	_, err := w.conn.Write(msg)
	if err != nil {
		return err
	}
	return nil
}
