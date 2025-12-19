package main

import (
	"bytes"
	"reflect"
	"testing"
)

func TestNewHandshake(t *testing.T) {
	var infoHash, peerID [20]byte
	copy(infoHash[:], []byte("test-infohash--------"))
	copy(peerID[:], []byte("test-peerid----------"))

	hs := NewHandshake(infoHash, peerID)

	if hs.Pstr != "BitTorrent protocol" {
		t.Errorf("NewHandshake() Pstr = %v, want 'BitTorrent protocol'", hs.Pstr)
	}
	if !reflect.DeepEqual(hs.InfoHash, infoHash) {
		t.Errorf("NewHandshake() InfoHash = %v, want %v", hs.InfoHash, infoHash)
	}
	if !reflect.DeepEqual(hs.PeerID, peerID) {
		t.Errorf("NewHandshake() PeerID = %v, want %v", hs.PeerID, peerID)
	}
}

func TestHandshakeSerialize(t *testing.T) {
	var infoHash, peerID [20]byte
	copy(infoHash[:], []byte("test-infohash--------"))
	copy(peerID[:], []byte("test-peerid----------"))

	hs := NewHandshake(infoHash, peerID)
	got := hs.Serialize()

	// Should be exactly 68 bytes
	if len(got) != 68 {
		t.Errorf("Serialize() length = %d, want 68", len(got))
	}

	// First byte should be 19 (length of "BitTorrent protocol")
	if got[0] != 19 {
		t.Errorf("Serialize() pstrLen = %d, want 19", got[0])
	}

	// Verify InfoHash is at correct position (bytes 28-47)
	if !bytes.Equal(got[28:48], infoHash[:]) {
		t.Errorf("Serialize() InfoHash not at correct position")
	}

	// Verify PeerID is at correct position (bytes 48-67)
	if !bytes.Equal(got[48:68], peerID[:]) {
		t.Errorf("Serialize() PeerID not at correct position")
	}
}

func TestReadHandshake(t *testing.T) {
	var infoHash, peerID [20]byte
	copy(infoHash[:], []byte("test-infohash--------"))
	copy(peerID[:], []byte("test-peerid----------"))

	hs := NewHandshake(infoHash, peerID)
	serialized := hs.Serialize()

	tests := []struct {
		name      string
		input     []byte
		want      *Handshake
		wantError bool
	}{
		{
			name:      "valid handshake round-trip",
			input:     serialized,
			want:      hs,
			wantError: false,
		},
		{
			name:      "invalid pstr length",
			input:     func() []byte { buf := make([]byte, 68); buf[0] = 20; return buf }(),
			want:      nil,
			wantError: true,
		},
		{
			name:      "short read",
			input:     []byte{1, 2, 3}, // Less than 68 bytes
			want:      nil,
			wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadHandshake(bytes.NewReader(tt.input))
			if (err != nil) != tt.wantError {
				t.Errorf("ReadHandshake() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadHandshake() = %v, want %v", got, tt.want)
			}
		})
	}
}
