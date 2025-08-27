package main

import (
	"TDMR87/protohackers/server"
	"bufio"
	"net"
	"testing"
)

func TestServer(t *testing.T) {
	t.Parallel()

	var testCases = map[string]struct {
		request  string
		response string
	}{
		"echo 1": {
			request:  "Hello, World!",
			response: "Hello, World!",
		},
		"echo 2": {
			request:  "Some test 123",
			response: "Some test 123",
		},
	}

	listener, err := server.StartListener(":0", handle)
	if err != nil {
		t.Fatal("Error starting server:", err)
	}

	defer listener.Close()

	for _, tt := range testCases {
		conn, err := net.Dial("tcp", listener.Addr().String())
		if err != nil {
			t.Fatal("Error connecting to server:", err)
		}

		defer conn.Close()

		_, err = conn.Write([]byte(tt.request + "\n"))
		if err != nil {
			t.Fatal("Error writing to server:", err)
		}

		scanner := bufio.NewScanner(conn)
		if scanner.Scan() {
			if scanner.Text() != tt.response {
				t.Errorf("Expected response %q, got %q", tt.response, scanner.Text())
			}
		} else if err := scanner.Err(); err != nil {
			t.Fatal("Error reading response from server:", err)
		} else {
			t.Fatal("No response from server")
		}
	}
}