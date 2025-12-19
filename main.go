package main

import "fmt"

func main() {
	// 1. Load torrent
	bt, err := Open("nuremberg.torrent")
	if err != nil {
		panic(err)
	}

	torrent, err := bt.ToTorrentFile()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Downloading: %s (%d bytes)\n", torrent.Name, torrent.Length)

	// 2. Start orchestration
	// starts workers in downloader.go
	err = torrent.Download()
	if err != nil {
		fmt.Printf("Download failed: %v\n", err)
		return
	}

	fmt.Println("Successfully downloaded: ", torrent.Name)
}
