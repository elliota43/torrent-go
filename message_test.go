package main

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

func TestMessageSerialize(t *testing.T) {
	tests := []struct {
		name string
		msg  *Message
		want []byte
	}{
		{
			name: "message with payload",
			msg:  &Message{ID: MsgUnchoke, Payload: []byte{1, 2, 3}},
			want: func() []byte {
				length := uint32(4) // 1 byte ID + 3 bytes payload
				buf := make([]byte, 4+length)
				binary.BigEndian.PutUint32(buf[0:4], length)
				buf[4] = byte(MsgUnchoke)
				copy(buf[5:], []byte{1, 2, 3})
				return buf
			}(),
		},
		{
			name: "message without payload",
			msg:  &Message{ID: MsgChoke, Payload: nil},
			want: func() []byte {
				length := uint32(1) // 1 byte ID
				buf := make([]byte, 4+length)
				binary.BigEndian.PutUint32(buf[0:4], length)
				buf[4] = byte(MsgChoke)
				return buf
			}(),
		},
		{
			name: "keep-alive",
			msg:  nil,
			want: []byte{0, 0, 0, 0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []byte
			if tt.msg == nil {
				got = (*Message)(nil).Serialize()
			} else {
				got = tt.msg.Serialize()
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Serialize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadMessage(t *testing.T) {
	tests := []struct {
		name      string
		input     []byte
		want      *Message
		wantError bool
	}{
		{
			name: "message with payload",
			input: func() []byte {
				msg := &Message{ID: MsgUnchoke, Payload: []byte{1, 2, 3}}
				return msg.Serialize()
			}(),
			want:      &Message{ID: MsgUnchoke, Payload: []byte{1, 2, 3}},
			wantError: false,
		},
		{
			name:      "keep-alive",
			input:     []byte{0, 0, 0, 0},
			want:      nil,
			wantError: false,
		},
		{
			name: "message too large",
			input: func() []byte {
				length := MaxMessageSize + 1
				buf := make([]byte, 4)
				binary.BigEndian.PutUint32(buf[0:4], uint32(length))
				return buf
			}(),
			want:      nil,
			wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadMessage(bytes.NewReader(tt.input))
			if (err != nil) != tt.wantError {
				t.Errorf("ReadMessage() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatRequest(t *testing.T) {
	tests := []struct {
		name   string
		index  int
		begin  int
		length int
		want   *Message
	}{
		{
			name:   "valid request",
			index:  5,
			begin:  16384,
			length: 16384,
			want: func() *Message {
				payload := make([]byte, 12)
				binary.BigEndian.PutUint32(payload[0:4], 5)
				binary.BigEndian.PutUint32(payload[4:8], 16384)
				binary.BigEndian.PutUint32(payload[8:12], 16384)
				return &Message{ID: MsgRequest, Payload: payload}
			}(),
		},
		{
			name:   "zero values",
			index:  0,
			begin:  0,
			length: 0,
			want: func() *Message {
				payload := make([]byte, 12)
				return &Message{ID: MsgRequest, Payload: payload}
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatRequest(tt.index, tt.begin, tt.length)
			if got.ID != tt.want.ID {
				t.Errorf("FormatRequest() ID = %v, want %v", got.ID, tt.want.ID)
			}
			if !reflect.DeepEqual(got.Payload, tt.want.Payload) {
				t.Errorf("FormatRequest() Payload = %v, want %v", got.Payload, tt.want.Payload)
			}
		})
	}
}
