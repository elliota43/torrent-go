package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"os"
	"testing"
)

func TestHandlePieceMsg(t *testing.T) {
	tests := []struct {
		name     string
		msg      *Message
		progress *pieceProgress
		wantBuf  []byte
		wantDown int
	}{
		{
			name: "valid piece message at beginning",
			msg: func() *Message {
				payload := make([]byte, 12+5)
				binary.BigEndian.PutUint32(payload[0:4], 0) // index
				binary.BigEndian.PutUint32(payload[4:8], 0) // begin
				copy(payload[8:13], []byte("hello"))        // block data (exactly 5 bytes)
				return &Message{ID: MsgPiece, Payload: payload}
			}(),
			progress: &pieceProgress{
				index:      0,
				buf:        make([]byte, 16384),
				downloaded: 0,
			},
			wantBuf:  func() []byte { b := make([]byte, 16384); copy(b[0:5], []byte("hello")); return b }(),
			wantDown: 5, // Block data is 5 bytes (payload[8:13] = 5 bytes)
		},
		{
			name: "valid piece message at offset",
			msg: func() *Message {
				payload := make([]byte, 12+4)
				binary.BigEndian.PutUint32(payload[0:4], 0)   // index
				binary.BigEndian.PutUint32(payload[4:8], 100) // begin
				copy(payload[8:12], []byte("test"))           // block data (exactly 4 bytes)
				return &Message{ID: MsgPiece, Payload: payload}
			}(),
			progress: &pieceProgress{
				index:      0,
				buf:        make([]byte, 16384),
				downloaded: 0,
			},
			wantBuf:  func() []byte { b := make([]byte, 16384); copy(b[100:104], []byte("test")); return b }(),
			wantDown: 4, // Block data is 4 bytes
		},
		{
			name: "multiple blocks",
			msg: func() *Message {
				payload := make([]byte, 12+3)
				binary.BigEndian.PutUint32(payload[0:4], 0) // index
				binary.BigEndian.PutUint32(payload[4:8], 0) // begin
				copy(payload[8:11], []byte("abc"))          // block data (exactly 3 bytes)
				return &Message{ID: MsgPiece, Payload: payload}
			}(),
			progress: &pieceProgress{
				index:      0,
				buf:        make([]byte, 16384),
				downloaded: 5, // Already downloaded 5 bytes
			},
			wantBuf:  func() []byte { b := make([]byte, 16384); copy(b[0:3], []byte("abc")); return b }(),
			wantDown: 8, // 5 + 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tf := &TorrentFile{}
			tf.handlePieceMsg(tt.msg, tt.progress)

			if !bytes.Equal(tt.progress.buf, tt.wantBuf) {
				t.Errorf("handlePieceMsg() buf = %v, want %v", tt.progress.buf[:len(tt.wantBuf)], tt.wantBuf)
			}
			if tt.progress.downloaded != tt.wantDown {
				t.Errorf("handlePieceMsg() downloaded = %d, want %d", tt.progress.downloaded, tt.wantDown)
			}
		})
	}
}

func TestHandlePieceMsg_BoundsCheck(t *testing.T) {
	// Test that out-of-bounds writes are handled gracefully
	msg := func() *Message {
		payload := make([]byte, 12+10)
		binary.BigEndian.PutUint32(payload[0:4], 0)     // index
		binary.BigEndian.PutUint32(payload[4:8], 16380) // begin (near end of buffer)
		copy(payload[8:], []byte("1234567890"))         // 10 bytes, would overflow
		return &Message{ID: MsgPiece, Payload: payload}
	}()

	progress := &pieceProgress{
		index:      0,
		buf:        make([]byte, 16384), // Only 16384 bytes
		downloaded: 0,
	}

	tf := &TorrentFile{}
	// Should not panic, but may not copy all data
	tf.handlePieceMsg(msg, progress)

	// Verify that only valid bytes were copied
	if progress.downloaded > len(progress.buf) {
		t.Errorf("handlePieceMsg() downloaded = %d, exceeds buffer size %d", progress.downloaded, len(progress.buf))
	}
}

func TestVerifyAndSave(t *testing.T) {
	// Create temporary file
	tmpfile, err := os.CreateTemp("", "test-piece-*.dat")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	tf := &TorrentFile{
		PieceLength: 16384,
	}

	testData := []byte("test piece data")
	hash := sha1.Sum(testData)

	tests := []struct {
		name      string
		pw        *pieceWork
		buf       []byte
		wantError bool
	}{
		{
			name: "valid piece",
			pw: &pieceWork{
				index:  0,
				hash:   hash,
				length: len(testData),
			},
			buf:       testData,
			wantError: false,
		},
		{
			name: "hash mismatch",
			pw: &pieceWork{
				index:  0,
				hash:   [20]byte{1, 2, 3}, // Wrong hash
				length: len(testData),
			},
			buf:       testData,
			wantError: true,
		},
		{
			name: "wrong piece index",
			pw: &pieceWork{
				index:  5, // Different index
				hash:   hash,
				length: len(testData),
			},
			buf:       testData,
			wantError: false, // Hash still matches, just wrong position
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := os.OpenFile(tmpfile.Name(), os.O_WRONLY|os.O_CREATE, 0666)
			if err != nil {
				t.Fatal(err)
			}

			err = tf.VerifyAndSave(tt.pw, tt.buf, file)
			file.Close()

			if (err != nil) != tt.wantError {
				t.Errorf("VerifyAndSave() error = %v, wantError %v", err, tt.wantError)
			}

			if !tt.wantError {
				// Verify data was written at correct offset
				file, err = os.Open(tmpfile.Name())
				if err != nil {
					t.Fatal(err)
				}
				readBuf := make([]byte, len(tt.buf))
				offset := int64(tt.pw.index * tf.PieceLength)
				file.ReadAt(readBuf, offset)
				file.Close()

				if !bytes.Equal(readBuf, tt.buf) {
					t.Errorf("VerifyAndSave() wrote incorrect data at offset %d", offset)
				}
			}
		})
	}
}
