package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"github.com/jackpal/bencode-go"
	"net/url"
	"os"
	"strconv"
)

type TorrentFile struct {
	Announce    string
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
}

func (t *TorrentFile) TrackerUrl(peerID [20]byte, port uint16) (string, error) {
	base, err := url.Parse(t.Announce)
	if err != nil {
		return "", err
	}

	params := url.Values{
		"info_hash":  []string{string(t.InfoHash[:])},
		"peer_id":    []string{string(peerID[:])},
		"port":       []string{strconv.Itoa(int(port))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(t.Length)},
	}

	base.RawQuery = params.Encode()
	return base.String(), nil
}

func (b *bencodeTorrent) ToTorrentFile() (TorrentFile, error) {
	const hashLen = 20
	piecesBinary := []byte(b.Info.Pieces)

	if len(piecesBinary)%hashLen != 0 {
		return TorrentFile{}, fmt.Errorf("invalid pieces length")
	}

	numPieces := len(piecesBinary) / hashLen

	hashes := make([][20]byte, numPieces)

	for i := 0; i < numPieces; i++ {
		copy(hashes[i][:], piecesBinary[i*hashLen:(i+1)*hashLen])
	}

	// Generate InfoHash
	var infoBuffer bytes.Buffer
	err := bencode.Marshal(&infoBuffer, b.Info)
	if err != nil {
		return TorrentFile{}, err
	}
	infoHash := sha1.Sum(infoBuffer.Bytes())

	return TorrentFile{
		Announce:    b.Announce,
		InfoHash:    infoHash,
		PieceHashes: hashes,
		PieceLength: b.Info.PieceLength,
		Length:      b.Info.Length,
		Name:        b.Info.Name,
	}, nil
}

type bencodeInfo struct {
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
}

type bencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     bencodeInfo `bencode:"info"`
}

func Open(path string) (*bencodeTorrent, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bt := bencodeTorrent{}
	err = bencode.Unmarshal(file, &bt)

	if err != nil {
		return nil, err
	}

	return &bt, nil
}
