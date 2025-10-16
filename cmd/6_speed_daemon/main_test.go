package main

import (
	"TDMR87/go_protohackers/internal/server"
	"net"
	"testing"
	"time"
)

func Test_WantHeartBeat_OnlyOnePerClientAllowed(t *testing.T) {
	listener, err := server.StartTcpListener(":0", handle)
	if err != nil {
		t.Fatal("Error starting server:", err)
	}
	defer listener.Close()

	conn, _ := net.Dial("tcp", listener.Addr().String())
	defer conn.Close()

	conn.Write(WantHeartBeat{Interval: 9999}.Encode())
	time.Sleep(100 * time.Millisecond)
	conn.Write(WantHeartBeat{Interval: 9999}.Encode())

	buf := make([]byte, Error{}.Size())
	n, _ := conn.Read(buf)

	response, err := Error{}.Decode(buf[:n])
	if err != nil {
		t.Fatal("Error decoding response:", err)
	}

	if response.Msg != "Heart beats are already being sent to this client" {
		t.Fatalf("expected error message 'Heart beats are already being sent to this client', got '%s'", response.Msg)
	}
}

func Test_WantHeartBeat_SendsHeartBeats(t *testing.T) {
	listener, err := server.StartTcpListener(":0", handle)
	if err != nil {
		t.Fatal("Error starting server:", err)
	}
	defer listener.Close()

	conn, _ := net.Dial("tcp", listener.Addr().String())
	defer conn.Close()

	// Request heartbeats every 100ms (1 decisecond)
	conn.Write(WantHeartBeat{Interval: 1}.Encode())

	// Read the first heartbeat
	buf := make([]byte, HeartBeat{}.Size())
	conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatal("Expected heartbeat, got error:", err)
	}

	// Got heartbeat, decode to verify
	_, err = HeartBeat{}.Decode(buf[:n])
	if err != nil {
		t.Fatal("Error decoding heartbeat:", err)
	}

	// Read second heartbeat to verify timing
	conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	n, err = conn.Read(buf)
	if err != nil {
		t.Fatal("Expected second heartbeat, got error:", err)
	}

	// Got second heartbeat, decode to verify
	_, err = HeartBeat{}.Decode(buf[:n])
	if err != nil {
		t.Fatal("Error decoding second heartbeat:", err)
	}
}

func Test_WantHeartBeat_ZeroIntervalNoHeartBeats(t *testing.T) {
	listener, err := server.StartTcpListener(":0", handle)
	if err != nil {
		t.Fatal("Error starting server:", err)
	}
	defer listener.Close()

	conn, _ := net.Dial("tcp", listener.Addr().String())
	defer conn.Close()

	// Request heartbeats with interval 0 (should not send heartbeats)
	conn.Write(WantHeartBeat{Interval: 0}.Encode())

	// Wait a bit for the server to process
	time.Sleep(100 * time.Millisecond)

	// Try to read with a short deadline - should timeout because no heartbeats sent
	buf := make([]byte, HeartBeat{}.Size())
	conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	_, err = conn.Read(buf)

	if err == nil {
		t.Fatal("Expected no heartbeat for interval 0, but got one")
	}
}