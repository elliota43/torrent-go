package main

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"os"
)

type pieceProgress struct {
	index      int
	buf        []byte
	downloaded int
}

func (t *TorrentFile) handlePieceMsg(msg *Message, progress *pieceProgress) {
	begin := binary.BigEndian.Uint32(msg.Payload[4:8])
	block := msg.Payload[8:]

	// Bounds check to prevent buffer overflow
	if int(begin)+len(block) > len(progress.buf) {
		return // Skip invalid block
	}

	copy(progress.buf[begin:], block)
	progress.downloaded += len(block)
}

func (t *TorrentFile) VerifyAndSave(pw *pieceWork, buf []byte, file *os.File) error {
	hash := sha1.Sum(buf)
	if hash != pw.hash {
		return fmt.Errorf("piece %d hash mismatch", pw.index)
	}

	offset := int64(pw.index * t.PieceLength)
	_, err := file.WriteAt(buf, offset)
	return err
}
