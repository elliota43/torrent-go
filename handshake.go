package main

import (
	"fmt"
	"io"
)

type Handshake struct {
	Pstr     string
	InfoHash [20]byte
	PeerID   [20]byte
}

func NewHandshake(infoHash, peerID [20]byte) *Handshake {
	return &Handshake{
		Pstr:     "BitTorrent protocol",
		InfoHash: infoHash,
		PeerID:   peerID,
	}
}

// Serialize turns the struct into a 68-byte buffer to send over TCP
func (h *Handshake) Serialize() []byte {
	buf := make([]byte, len(h.Pstr)+49)
	buf[0] = byte(len(h.Pstr))
	curr := 1
	curr += copy(buf[curr:], h.Pstr)
	curr += copy(buf[curr:], make([]byte, 8))
	curr += copy(buf[curr:], h.InfoHash[:])
	curr += copy(buf[curr:], h.PeerID[:])
	return buf
}

func ReadHandshake(r io.Reader) (*Handshake, error) {
	buf := make([]byte, 68)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}

	pstrLen := int(buf[0])
	if pstrLen != 19 {
		return nil, fmt.Errorf("invalid pstrLen: %d", pstrLen)
	}

	var infoHash, peerID [20]byte
	copy(infoHash[:], buf[28:48])
	copy(peerID[:], buf[48:68])

	return &Handshake{
		Pstr:     string(buf[1:20]),
		InfoHash: infoHash,
		PeerID:   peerID,
	}, nil
}
