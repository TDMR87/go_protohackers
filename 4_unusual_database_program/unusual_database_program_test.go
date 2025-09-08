package main

import (
	"TDMR87/protohackers/server"
	"net"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	listener, _ := server.StartUdpListener(":8080", handle)
	defer listener.Close()

	serverAddr, err := net.ResolveUDPAddr("udp", listener.LocalAddr().String())
    if err != nil {
        t.Fatal("Connecting to the server failed", err)
    }

	conn, err := net.DialUDP("udp", nil, serverAddr)
    if err != nil {
        t.Fatal("Connecting to the server failed", err)
    }
    defer conn.Close()

    // Insert
    conn.Write([]byte("message=hello"))
	time.Sleep(50 * time.Millisecond)

	// Get
	conn.Write([]byte("message"))
	buf := make([]byte, 1000)
	n, _, _ := conn.ReadFromUDP(buf)
	response := string(buf[:n])
	if response != "message=hello" {
		t.Fatalf("Expected 'message=hello', got %s", response)
	}
}

func TestNonExistentKey(t *testing.T) {
	listener, _ := server.StartUdpListener(":8080", handle)
	defer listener.Close()

	serverAddr, err := net.ResolveUDPAddr("udp", listener.LocalAddr().String())
    if err != nil {
        t.Fatal("Connecting to the server failed", err)
    }

	conn, err := net.DialUDP("udp", nil, serverAddr)
    if err != nil {
        t.Fatal("Connecting to the server failed", err)
    }
    defer conn.Close()

	conn.Write([]byte("foobar"))
	buf := make([]byte, 1000)
	n, _, _ := conn.ReadFromUDP(buf)
	response := string(buf[:n])
	if response != "foobar=" {
		t.Fatalf("Expected empty string, got %s", response)
	}
}

func TestRetrieveVersion(t *testing.T) {
	listener, _ := server.StartUdpListener(":8080", handle)
	defer listener.Close()

	serverAddr, err := net.ResolveUDPAddr("udp", listener.LocalAddr().String())
    if err != nil {
        t.Fatal("Connecting to the server failed", err)
    }

	conn, err := net.DialUDP("udp", nil, serverAddr)
    if err != nil {
        t.Fatal("Connecting to the server failed", err)
    }
    defer conn.Close()

	conn.Write([]byte("version"))
	buf := make([]byte, 1000)
	n, _, _ := conn.ReadFromUDP(buf)
	response := string(buf[:n])
	if response != "version=6.6.6" {
		t.Fatalf("Expected version=6.6.6, got %s", response)
	}
}

func TestInsertVersion(t *testing.T) {
	listener, _ := server.StartUdpListener(":8080", handle)
	defer listener.Close()

	serverAddr, err := net.ResolveUDPAddr("udp", listener.LocalAddr().String())
    if err != nil {
        t.Fatal("Connecting to the server failed", err)
    }

	conn, err := net.DialUDP("udp", nil, serverAddr)
    if err != nil {
        t.Fatal("Connecting to the server failed", err)
    }
    defer conn.Close()

	// Try to modify the version
	conn.Write([]byte("version=9.9.9"))

	// Get the version
	conn.Write([]byte("version"))
	buf := make([]byte, 1000)
	n, _, _ := conn.ReadFromUDP(buf)
	response := string(buf[:n])
	if response != "6.6.6" {
		t.Fatalf("Expected 6.6.6, got %s", response)
	}
}

func TestParseMessage(t *testing.T) {
	testCases := map[string]struct{
		Input string
		ExpectedKey string
		ExpectedVal string
	}{
		"case 1":
		{
			Input: "foo=bar",
			ExpectedKey: "foo",
			ExpectedVal: "bar",
		},
		"case 2":
		{
			Input: "foo=bar=baz",
			ExpectedKey: "foo",
			ExpectedVal: "bar=baz",	
		},
		"case 3":
		{
			Input: "foo=",
			ExpectedKey: "foo",
			ExpectedVal: "",
		},
		"case 4":
		{
			Input: "foo===",
			ExpectedKey: "foo",
			ExpectedVal: "==",
		},
		"case 5":
		{
			Input: "=foo",
			ExpectedKey: "",
			ExpectedVal: "foo",
		},
	}

	for _, tt := range testCases {
		key, val := parse(tt.Input)
		if key != tt.ExpectedKey {
			t.Fatalf("Expected key %s, got %s", tt.ExpectedKey, key)
		}
		if val != tt.ExpectedVal {
			t.Fatalf("Expected value %s, got %s", tt.ExpectedVal, val)
		}
	}
}