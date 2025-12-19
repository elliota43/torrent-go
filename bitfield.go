package main

type Bitfield []byte

// HasPiece checks if a bitfield has a specific piece index set
func (bf Bitfield) HasPiece(index int) bool {
	byteIndex := index / 8
	offset := index % 8
	if byteIndex < 0 || byteIndex >= len(bf) {
		return false
	}

	// Big-endian bit order the most significant bit is index 0
	return bf[byteIndex]>>(7-offset)&1 != 0
}
