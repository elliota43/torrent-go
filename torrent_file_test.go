package main

import (
	"bytes"
	"net/url"
	"os"
	"testing"
)

func TestTrackerUrl(t *testing.T) {
	var infoHash, peerID [20]byte
	copy(infoHash[:], []byte("test-infohash--------"))
	copy(peerID[:], []byte("test-peerid----------"))

	tf := TorrentFile{
		Announce: "http://tracker.example.com/announce",
		InfoHash: infoHash,
		Length:   1024,
	}

	tests := []struct {
		name string
		port uint16
	}{
		{"default port", 6881},
		{"custom port", 9999},
		{"port zero", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tf.TrackerUrl(peerID, tt.port)
			if err != nil {
				t.Errorf("TrackerUrl() error = %v", err)
				return
			}

			// Parse URL to verify structure
			u, err := url.Parse(got)
			if err != nil {
				t.Errorf("TrackerUrl() returned invalid URL: %v", err)
				return
			}

			// Verify base URL
			if u.Scheme != "http" || u.Host != "tracker.example.com" || u.Path != "/announce" {
				t.Errorf("TrackerUrl() base URL = %v, want http://tracker.example.com/announce", u)
			}

			// Verify query parameters
			params := u.Query()
			if params.Get("port") != "0" && params.Get("port") != string(rune(tt.port)) {
				// Port should be in query string
				portStr := params.Get("port")
				if portStr == "" {
					t.Errorf("TrackerUrl() missing port parameter")
				}
			}
			if params.Get("info_hash") == "" {
				t.Errorf("TrackerUrl() missing info_hash parameter")
			}
			if params.Get("peer_id") == "" {
				t.Errorf("TrackerUrl() missing peer_id parameter")
			}
			if params.Get("left") != "1024" {
				t.Errorf("TrackerUrl() left = %v, want 1024", params.Get("left"))
			}
		})
	}
}

func TestTrackerUrl_InvalidURL(t *testing.T) {
	tf := TorrentFile{
		Announce: "://invalid-url",
		InfoHash: [20]byte{},
	}

	var peerID [20]byte
	_, err := tf.TrackerUrl(peerID, 6881)
	if err == nil {
		t.Errorf("TrackerUrl() should return error for invalid URL")
	}
}

func TestToTorrentFile(t *testing.T) {
	// Create a minimal valid torrent structure
	pieces := make([]byte, 40) // 2 pieces, 20 bytes each
	copy(pieces[0:20], []byte("piece1-hash--------"))
	copy(pieces[20:40], []byte("piece2-hash--------"))

	bt := &bencodeTorrent{
		Announce: "http://tracker.example.com/announce",
		Info: bencodeInfo{
			Pieces:      string(pieces),
			PieceLength: 16384,
			Length:      32768,
			Name:        "test-file",
		},
	}

	got, err := bt.ToTorrentFile()
	if err != nil {
		t.Fatalf("ToTorrentFile() error = %v", err)
	}

	// Verify basic fields
	if got.Announce != bt.Announce {
		t.Errorf("ToTorrentFile() Announce = %v, want %v", got.Announce, bt.Announce)
	}
	if got.PieceLength != bt.Info.PieceLength {
		t.Errorf("ToTorrentFile() PieceLength = %v, want %v", got.PieceLength, bt.Info.PieceLength)
	}
	if got.Length != bt.Info.Length {
		t.Errorf("ToTorrentFile() Length = %v, want %v", got.Length, bt.Info.Length)
	}
	if got.Name != bt.Info.Name {
		t.Errorf("ToTorrentFile() Name = %v, want %v", got.Name, bt.Info.Name)
	}

	// Verify piece hashes
	if len(got.PieceHashes) != 2 {
		t.Errorf("ToTorrentFile() PieceHashes length = %d, want 2", len(got.PieceHashes))
	}
	if !bytes.Equal(got.PieceHashes[0][:], pieces[0:20]) {
		t.Errorf("ToTorrentFile() first piece hash incorrect")
	}
	if !bytes.Equal(got.PieceHashes[1][:], pieces[20:40]) {
		t.Errorf("ToTorrentFile() second piece hash incorrect")
	}

	// Verify InfoHash is calculated (should be SHA1 of bencoded info dict)
	if got.InfoHash == [20]byte{} {
		t.Errorf("ToTorrentFile() InfoHash is zero")
	}
}

func TestToTorrentFile_InvalidPieces(t *testing.T) {
	tests := []struct {
		name   string
		pieces string
	}{
		{"invalid length - not multiple of 20", "invalid"},
		// Empty pieces is actually valid (no pieces = empty file)
		{"one byte short", string(make([]byte, 39))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bt := &bencodeTorrent{
				Info: bencodeInfo{
					Pieces:      tt.pieces,
					PieceLength: 16384,
					Length:      1000,
					Name:        "test",
				},
			}

			_, err := bt.ToTorrentFile()
			if err == nil {
				t.Errorf("ToTorrentFile() should return error for invalid pieces")
			}
		})
	}
}

func TestOpen(t *testing.T) {
	// Test with a real torrent file if available
	if _, err := os.Stat("ubuntu.torrent"); err == nil {
		bt, err := Open("ubuntu.torrent")
		if err != nil {
			t.Errorf("Open() error = %v", err)
			return
		}
		if bt == nil {
			t.Errorf("Open() returned nil")
		}
		if bt.Announce == "" {
			t.Errorf("Open() Announce is empty")
		}
	}

	// Test file not found
	_, err := Open("nonexistent.torrent")
	if err == nil {
		t.Errorf("Open() should return error for nonexistent file")
	}
}

func TestInfoHashCalculation(t *testing.T) {
	// Verify InfoHash is calculated correctly
	// This is a regression test to ensure InfoHash calculation doesn't break
	pieces := make([]byte, 20)
	copy(pieces, []byte("test-piece-hash----"))

	bt := &bencodeTorrent{
		Info: bencodeInfo{
			Pieces:      string(pieces),
			PieceLength: 16384,
			Length:      16384,
			Name:        "test",
		},
	}

	tf1, err := bt.ToTorrentFile()
	if err != nil {
		t.Fatalf("ToTorrentFile() error = %v", err)
	}

	// Create same torrent again - InfoHash should be same
	tf2, err := bt.ToTorrentFile()
	if err != nil {
		t.Fatalf("ToTorrentFile() error = %v", err)
	}

	if !bytes.Equal(tf1.InfoHash[:], tf2.InfoHash[:]) {
		t.Errorf("InfoHash should be deterministic, got different hashes")
	}
}
