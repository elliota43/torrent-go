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
	begin := binary.BigEndian.Uint16(msg.Payload[4:8])
	block := msg.Payload[8:]

	copy(progress.buf[begin:], block)
	progress.downloaded += len(block)
}

func (t *TorrentFile) VerifyAndSave(pw *pieceWork, buf []byte) error {
	hash := sha1.Sum(buf)
	if hash != pw.hash {
		return fmt.Errorf("piece %d hash mismatch", pw.index)
	}

	offset := int64(pw.index * t.PieceLength)
	f, err := os.OpenFile(t.Name, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = f.WriteAt(buf, offset)
	return err
}
