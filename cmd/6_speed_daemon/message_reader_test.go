package main

import (
	"bytes"
	"testing"
)

func TestMessageReader_WantHeartBeat(t *testing.T) {
	testCases := map[string]struct {
		byteStream       []byte
		expectedInterval []uint32
		wantErr          bool
	}{
		"single message in one read": {
			byteStream:       WantHeartBeat{Interval: 10}.Encode(),
			expectedInterval: []uint32{10},
		},
		"two messages concatenated": {
			byteStream: append(
				WantHeartBeat{Interval: 20}.Encode(),
				WantHeartBeat{Interval: 30}.Encode()...,
			),
			expectedInterval: []uint32{20, 30},
		},
	}

	for _, tt := range testCases {
		conn := bytes.NewBuffer(tt.byteStream)
		reader := NewMessageReader(conn)

		for _, expected := range tt.expectedInterval {
			msg, err := reader.NextMessage()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			hb, ok := msg.(WantHeartBeat)

			if !ok {
				t.Fatalf("expected WantHeartBeat, got %T", msg)
			}
			if hb.Interval != expected {
				t.Fatalf("expected interval %d, got %d", expected, hb.Interval)
			}
		}

		_, err := reader.NextMessage()
		if err == nil {
			t.Fatalf("expected error after all messages consumed, got nil")
		}
	}
}
