package main

import "testing"

func TestHasPiece(t *testing.T) {
	tests := []struct {
		name     string
		bf       Bitfield
		index    int
		expected bool
	}{
		{
			name:     "single byte - piece 4 set",
			bf:       Bitfield{0b00001010}, // 00001010 -> Pieces 4 and 6
			index:    4,
			expected: true,
		},
		{
			name:     "single byte - piece 6 set",
			bf:       Bitfield{0b00001010},
			index:    6,
			expected: true,
		},
		{
			name:     "single byte - piece 0 not set",
			bf:       Bitfield{0b00001010},
			index:    0,
			expected: false,
		},
		{
			name:     "single byte - piece 7 not set",
			bf:       Bitfield{0b00001010},
			index:    7,
			expected: false,
		},
		{
			name:     "out of bounds",
			bf:       Bitfield{0b00001010},
			index:    10,
			expected: false,
		},
		{
			name:     "multi-byte - piece 7 (first byte MSB)",
			bf:       Bitfield{0x01, 0x80}, // 00000001 10000000
			index:    7,
			expected: true,
		},
		{
			name:     "multi-byte - piece 8 (second byte LSB)",
			bf:       Bitfield{0x01, 0x80}, // 00000001 10000000 - piece 8 is LSB of second byte
			index:    8,
			expected: true,
		},
		{
			name:     "empty bitfield",
			bf:       Bitfield{},
			index:    0,
			expected: false,
		},
		{
			name:     "negative index",
			bf:       Bitfield{0b00001010},
			index:    -1,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bf.HasPiece(tt.index); got != tt.expected {
				t.Errorf("HasPiece(%d) = %v, want %v", tt.index, got, tt.expected)
			}
		})
	}
}
