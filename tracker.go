package main

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	"github.com/jackpal/bencode-go"
)

type bencodeTrackerResponse struct {
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"`
}

func (t *TorrentFile) RequestPeers(peerID [20]byte, port uint16) ([]Peer, error) {
	url, err := t.TrackerUrl(peerID, port)
	if err != nil {
		return nil, err
	}

	c := &http.Client{Timeout: 15 * time.Second}
	resp, err := c.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tracker returned status code %d: %s", resp.StatusCode, resp.Status)
	}

	trackerResp := bencodeTrackerResponse{}
	err = bencode.Unmarshal(resp.Body, &trackerResp)
	if err != nil {
		return nil, err
	}

	return UnmarshalPeer([]byte(trackerResp.Peers))
}

func GeneratePeerID() ([20]byte, error) {
	var id [20]byte

	prefix := []byte("-GM1000-")
	copy(id[:], prefix)

	_, err := rand.Read(id[len(prefix):])
	return id, err
}
