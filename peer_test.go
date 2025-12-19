package main

import (
	"net"
	"testing"
)

func TestUnmarshalPeer(t *testing.T) {
	tests := []struct {
		name      string
		input     []byte
		want      []Peer
		wantError bool
	}{
		{
			name:      "single peer",
			input:     []byte{192, 168, 1, 1, 0x1A, 0xE1}, // IP: 192.168.1.1, Port: 6881
			want:      []Peer{{IP: net.IPv4(192, 168, 1, 1), Port: 6881}},
			wantError: false,
		},
		{
			name: "multiple peers",
			input: []byte{
				192, 168, 1, 1, 0x1A, 0xE1, // 192.168.1.1:6881
				10, 0, 0, 1, 0x1A, 0xE2, // 10.0.0.1:6882
			},
			want: []Peer{
				{IP: net.IPv4(192, 168, 1, 1), Port: 6881},
				{IP: net.IPv4(10, 0, 0, 1), Port: 6882},
			},
			wantError: false,
		},
		{
			name:      "empty input",
			input:     []byte{},
			want:      []Peer{},
			wantError: false,
		},
		{
			name:      "invalid length - 5 bytes",
			input:     []byte{1, 2, 3, 4, 5},
			want:      nil,
			wantError: true,
		},
		{
			name:      "invalid length - 7 bytes",
			input:     []byte{1, 2, 3, 4, 5, 6, 7},
			want:      nil,
			wantError: true,
		},
		{
			name:      "port zero",
			input:     []byte{192, 168, 1, 1, 0x00, 0x00},
			want:      []Peer{{IP: net.IPv4(192, 168, 1, 1), Port: 0}},
			wantError: false,
		},
		{
			name:      "maximum port",
			input:     []byte{192, 168, 1, 1, 0xFF, 0xFF},
			want:      []Peer{{IP: net.IPv4(192, 168, 1, 1), Port: 65535}},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UnmarshalPeer(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("UnmarshalPeer() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("UnmarshalPeer() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if !got[i].IP.Equal(tt.want[i].IP) {
					t.Errorf("UnmarshalPeer() peer[%d].IP = %v, want %v", i, got[i].IP, tt.want[i].IP)
				}
				if got[i].Port != tt.want[i].Port {
					t.Errorf("UnmarshalPeer() peer[%d].Port = %v, want %v", i, got[i].Port, tt.want[i].Port)
				}
			}
		})
	}
}
