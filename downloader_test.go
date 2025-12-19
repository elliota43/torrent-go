package main

import (
	"encoding/binary"
	"net"
	"testing"
)

func TestAttemptDownloadPiece(t *testing.T) {
	// Create a pipe to simulate connection
	clientConn, serverConn := net.Pipe()

	pieceIndex := 0
	pieceLength := 16384
	blockSize := 16384

	// Simulate peer sending piece data
	go func() {
		// Read request message
		buf := make([]byte, 17) // 4 bytes length + 1 byte ID + 12 bytes payload
		serverConn.Read(buf)

		// Send piece message
		payload := make([]byte, 12+blockSize)
		binary.BigEndian.PutUint32(payload[0:4], uint32(pieceIndex))
		binary.BigEndian.PutUint32(payload[4:8], 0) // begin
		// Fill with test data
		for i := 0; i < blockSize; i++ {
			payload[8+i] = byte(i % 256)
		}

		msg := &Message{ID: MsgPiece, Payload: payload}
		serverConn.Write(msg.Serialize())
		serverConn.Close()
	}()

	tf := &TorrentFile{PieceLength: 16384}
	pw := &pieceWork{
		index:  pieceIndex,
		hash:   [20]byte{},
		length: pieceLength,
	}

	buf, err := tf.attemptDownloadPiece(clientConn, pw)
	clientConn.Close()

	if err != nil {
		t.Errorf("attemptDownloadPiece() error = %v", err)
		return
	}

	if len(buf) != pieceLength {
		t.Errorf("attemptDownloadPiece() length = %d, want %d", len(buf), pieceLength)
	}

	// Verify data
	for i := 0; i < blockSize; i++ {
		if buf[i] != byte(i%256) {
			t.Errorf("attemptDownloadPiece() data mismatch at index %d", i)
			break
		}
	}
}

func TestAttemptDownloadPiece_MultipleBlocks(t *testing.T) {
	clientConn, serverConn := net.Pipe()

	pieceIndex := 0
	pieceLength := 32768 // 2 blocks
	blockSize := 16384

	go func() {
		// Read first request
		buf := make([]byte, 17)
		serverConn.Read(buf)

		// Send first block
		payload1 := make([]byte, 12+blockSize)
		binary.BigEndian.PutUint32(payload1[0:4], uint32(pieceIndex))
		binary.BigEndian.PutUint32(payload1[4:8], 0)
		for i := 0; i < blockSize; i++ {
			payload1[8+i] = byte(i % 256)
		}
		msg1 := &Message{ID: MsgPiece, Payload: payload1}
		serverConn.Write(msg1.Serialize())

		// Read second request
		serverConn.Read(buf)

		// Send second block
		payload2 := make([]byte, 12+blockSize)
		binary.BigEndian.PutUint32(payload2[0:4], uint32(pieceIndex))
		binary.BigEndian.PutUint32(payload2[4:8], uint32(blockSize))
		for i := 0; i < blockSize; i++ {
			payload2[8+i] = byte((i + blockSize) % 256)
		}
		msg2 := &Message{ID: MsgPiece, Payload: payload2}
		serverConn.Write(msg2.Serialize())
		serverConn.Close()
	}()

	tf := &TorrentFile{PieceLength: 16384}
	pw := &pieceWork{
		index:  pieceIndex,
		hash:   [20]byte{},
		length: pieceLength,
	}

	buf, err := tf.attemptDownloadPiece(clientConn, pw)
	clientConn.Close()

	if err != nil {
		t.Errorf("attemptDownloadPiece() error = %v", err)
		return
	}

	if len(buf) != pieceLength {
		t.Errorf("attemptDownloadPiece() length = %d, want %d", len(buf), pieceLength)
	}
}

func TestAttemptDownloadPiece_ChokeMessage(t *testing.T) {
	clientConn, serverConn := net.Pipe()

	go func() {
		defer serverConn.Close()

		// Read request
		buf := make([]byte, 17)
		serverConn.Read(buf)

		// Send choke instead of piece
		chokeMsg := &Message{ID: MsgChoke, Payload: nil}
		serverConn.Write(chokeMsg.Serialize())
	}()

	tf := &TorrentFile{PieceLength: 16384}
	pw := &pieceWork{
		index:  0,
		hash:   [20]byte{},
		length: 16384,
	}

	_, err := tf.attemptDownloadPiece(clientConn, pw)
	clientConn.Close()

	if err == nil {
		t.Errorf("attemptDownloadPiece() should return error on choke")
	}
}

func TestAttemptDownloadPiece_ConnectionError(t *testing.T) {
	clientConn, serverConn := net.Pipe()

	// Close server immediately
	serverConn.Close()

	tf := &TorrentFile{PieceLength: 16384}
	pw := &pieceWork{
		index:  0,
		hash:   [20]byte{},
		length: 16384,
	}

	_, err := tf.attemptDownloadPiece(clientConn, pw)
	clientConn.Close()

	if err == nil {
		t.Errorf("attemptDownloadPiece() should return error on connection close")
	}
}
