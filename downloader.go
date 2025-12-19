package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const MaxBlockSize = 16384 // 16KB
type pieceWork struct {
	index  int
	hash   [20]byte
	length int
}

type pieceResult struct {
	index int
	buf   []byte
}

// safelyRequeueWork attempts to requeue work, handling closed channels gracefully
func safelyRequeueWork(work chan *pieceWork, pw *pieceWork) {
	defer func() {
		if recover() != nil {
			// Channel is closed, ignore the requeue attempt
		}
	}()
	select {
	case work <- pw:
	default:
		// Channel is full, work will be lost but that's acceptable
	}
}

func (t *TorrentFile) Download() error {
	peerID, err := GeneratePeerID()
	if err != nil {
		return fmt.Errorf("failed to generate peer ID: %v", err)
	}
	peers, err := t.RequestPeers(peerID, 6881)
	if err != nil {
		return fmt.Errorf("failed to request peers: %v", err)
	}

	if len(peers) == 0 {
		return fmt.Errorf("no peers available")
	}

	// Open the file once for all piece writes
	file, err := os.OpenFile(t.Name, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Make workQueue 2x size to allow requeuing failed work without blocking
	workQueue := make(chan *pieceWork, len(t.PieceHashes)*2)
	results := make(chan *pieceResult)

	// fill work queue
	for i, hash := range t.PieceHashes {
		pieceLength := t.PieceLength
		// Last piece might be shorter
		if i == len(t.PieceHashes)-1 {
			pieceLength = t.Length - (i * t.PieceLength)
		}
		workQueue <- &pieceWork{i, hash, pieceLength}
	}

	// start workers with WaitGroup tracking
	var wg sync.WaitGroup
	for _, peer := range peers {
		wg.Add(1)
		go func(p Peer) {
			defer wg.Done()
			t.startWorker(p, peerID, workQueue, results, file)
		}(peer)
	}

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// progress bar logic
	doneCount := 0
	totalPieces := len(t.PieceHashes)
	fmt.Printf("Downloading %s...\n", t.Name)
	for doneCount < totalPieces {
		_, ok := <-results
		if !ok {
			// Channel closed, all workers are done
			if doneCount < totalPieces {
				return fmt.Errorf("all workers finished but only %d/%d pieces downloaded", doneCount, totalPieces)
			}
			break
		}
		doneCount++

		percent := float64(doneCount) / float64(totalPieces) * 100
		fmt.Printf("\r[%-50s] %0.2f%% (%d/%d pieces)", strings.Repeat("-", int(percent/2)), percent, doneCount, totalPieces)
	}

	// All pieces downloaded, close workQueue so workers can exit
	// Use recover in safelyRequeueWork to handle any requeue attempts after closure
	close(workQueue)

	fmt.Println("\nDownload complete!")
	return nil
}

func (t *TorrentFile) startWorker(peer Peer, peerID [20]byte, work chan *pieceWork, results chan *pieceResult, file *os.File) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", peer.IP, peer.Port), 5*time.Second)
	if err != nil {
		return
	}
	defer conn.Close()

	// 1. Handshake
	hs := NewHandshake(t.InfoHash, peerID)
	if _, err := conn.Write(hs.Serialize()); err != nil {
		return
	}
	_, err = ReadHandshake(conn)
	if err != nil {
		return
	}

	// 2. Identify what the peer has (Bitfield)
	var bf Bitfield
	msg, err := ReadMessage(conn)
	if err == nil && msg.ID == MsgBitfield {
		bf = msg.Payload
	}

	// 3. Initialize Session State
	unchoked := false
	interestedMsg := (&Message{ID: MsgInterested}).Serialize()
	if _, err := conn.Write(interestedMsg); err != nil {
		return
	}

	// Set read deadline for unchoke waiting (30 second timeout)
	unchokeTimeout := 30 * time.Second

	// 4. THE CONSOLIDATED LOOP
	for pw := range work {
		// Skip peers that don't have our piece
		if bf != nil && !bf.HasPiece(pw.index) {
			safelyRequeueWork(work, pw)
			continue
		}

		// Wait for Unchoke with timeout
		for !unchoked {
			conn.SetReadDeadline(time.Now().Add(unchokeTimeout))
			msg, err := ReadMessage(conn)
			if err != nil {
				safelyRequeueWork(work, pw)
				return
			}
			if msg == nil {
				continue // Keep-alive message
			}
			switch msg.ID {
			case MsgUnchoke:
				unchoked = true
				conn.SetReadDeadline(time.Time{}) // Clear deadline
			case MsgChoke:
				unchoked = false // Peer choked us
			case MsgHave, MsgBitfield, MsgPiece:
				// Handle other messages but continue waiting for unchoke
				continue
			}
		}

		// 5. Download & Verify
		buf, err := t.attemptDownloadPiece(conn, pw)
		if err != nil {
			safelyRequeueWork(work, pw)
			return // Peer failed us, kill worker
		}

		if err := t.VerifyAndSave(pw, buf, file); err != nil {
			safelyRequeueWork(work, pw)
			continue
		}

		results <- &pieceResult{pw.index, buf}
	}
}

func (t *TorrentFile) attemptDownloadPiece(conn net.Conn, pw *pieceWork) ([]byte, error) {
	progress := pieceProgress{
		index: pw.index,
		buf:   make([]byte, pw.length),
	}

	for progress.downloaded < pw.length {
		blockSize := MaxBlockSize
		if pw.length-progress.downloaded < blockSize {
			blockSize = pw.length - progress.downloaded
		}

		// send request message
		req := FormatRequest(pw.index, progress.downloaded, blockSize)
		if _, err := conn.Write(req.Serialize()); err != nil {
			return nil, err
		}

		// Read piece message block
		msg, err := ReadMessage(conn)
		if err != nil {
			return nil, err
		}
		if msg == nil {
			continue // Keep-alive message
		}
		if msg.ID == MsgChoke {
			return nil, fmt.Errorf("peer choked during piece download")
		}
		if msg.ID != MsgPiece {
			continue
		}

		// add block to our buffer
		t.handlePieceMsg(msg, &progress)
	}

	return progress.buf, nil
}
