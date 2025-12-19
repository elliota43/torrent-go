package main

import (
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGeneratePeerID(t *testing.T) {
	tests := []struct {
		name      string
		wantError bool
	}{
		{"valid peer ID", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GeneratePeerID()
			if (err != nil) != tt.wantError {
				t.Errorf("GeneratePeerID() error = %v, wantError %v", err, tt.wantError)
				return
			}

			// Verify length
			if len(got) != 20 {
				t.Errorf("GeneratePeerID() length = %d, want 20", len(got))
			}

			// Verify prefix
			prefix := []byte("-GM1000-")
			if !bytes.Equal(got[:len(prefix)], prefix) {
				t.Errorf("GeneratePeerID() prefix = %v, want %v", got[:len(prefix)], prefix)
			}

			// Verify randomness - generate multiple IDs and check they're different
			id1, _ := GeneratePeerID()
			id2, _ := GeneratePeerID()
			if bytes.Equal(id1[:], id2[:]) {
				t.Errorf("GeneratePeerID() should generate different IDs")
			}
		})
	}
}

func TestRequestPeers(t *testing.T) {
	// Create a mock tracker server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "GET" {
			t.Errorf("RequestPeers() method = %v, want GET", r.Method)
		}

		// Create a valid tracker response
		// For simplicity, we'll return a bencoded response manually
		// In a real test, you'd use the bencode library
		response := "d8:intervali1800e5:peers6:" + string([]byte{192, 168, 1, 1, 0x1A, 0xE1}) + "e"
		w.Write([]byte(response))
	}))
	defer server.Close()

	tf := TorrentFile{
		Announce: server.URL,
		InfoHash: [20]byte{1, 2, 3},
		Length:   1024,
	}

	var peerID [20]byte
	peers, err := tf.RequestPeers(peerID, 6881)
	if err != nil {
		t.Errorf("RequestPeers() error = %v", err)
		return
	}

	if len(peers) == 0 {
		t.Errorf("RequestPeers() returned no peers")
	}
}

func TestRequestPeers_HTTPError(t *testing.T) {
	// Create a server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tf := TorrentFile{
		Announce: server.URL,
		InfoHash: [20]byte{},
	}

	var peerID [20]byte
	_, err := tf.RequestPeers(peerID, 6881)
	if err == nil {
		t.Errorf("RequestPeers() should return error for 404 status")
	}
}

func TestRequestPeers_InvalidResponse(t *testing.T) {
	// Create a server that returns invalid bencode
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("invalid bencode"))
	}))
	defer server.Close()

	tf := TorrentFile{
		Announce: server.URL,
		InfoHash: [20]byte{},
	}

	var peerID [20]byte
	_, err := tf.RequestPeers(peerID, 6881)
	if err == nil {
		t.Errorf("RequestPeers() should return error for invalid bencode")
	}
}

func TestRequestPeers_NetworkError(t *testing.T) {
	// Use an invalid URL to simulate network error
	tf := TorrentFile{
		Announce: "http://invalid-host-that-does-not-exist.local:9999/announce",
		InfoHash: [20]byte{},
	}

	var peerID [20]byte
	_, err := tf.RequestPeers(peerID, 6881)
	if err == nil {
		t.Errorf("RequestPeers() should return error for network failure")
	}
}

func TestRequestPeers_ValidBencodeResponse(t *testing.T) {
	// Create a proper bencoded response
	// d = dictionary, i = integer, 5:peers = string "peers" of length 5
	// We need to manually construct a valid bencode response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create response: d8:intervali1800e5:peers6:...e
		var buf bytes.Buffer
		buf.WriteString("d")          // dictionary start
		buf.WriteString("8:interval") // key "interval"
		buf.WriteString("i1800e")     // value 1800
		buf.WriteString("5:peers")    // key "peers"
		buf.WriteString("6:")         // value string of length 6
		peerData := []byte{192, 168, 1, 1, 0x1A, 0xE1}
		buf.Write(peerData)
		buf.WriteString("e") // dictionary end
		w.Write(buf.Bytes())
	}))
	defer server.Close()

	tf := TorrentFile{
		Announce: server.URL,
		InfoHash: [20]byte{},
		Length:   1024,
	}

	var peerID [20]byte
	peers, err := tf.RequestPeers(peerID, 6881)
	if err != nil {
		t.Errorf("RequestPeers() error = %v", err)
		return
	}

	if len(peers) != 1 {
		t.Errorf("RequestPeers() returned %d peers, want 1", len(peers))
		return
	}

	// Verify peer data
	expectedIP := net.IPv4(192, 168, 1, 1)
	if !peers[0].IP.Equal(expectedIP) {
		t.Errorf("RequestPeers() peer IP = %v, want %v", peers[0].IP, expectedIP)
	}

	expectedPort := uint16(6881)
	if peers[0].Port != expectedPort {
		t.Errorf("RequestPeers() peer Port = %v, want %v", peers[0].Port, expectedPort)
	}
}
