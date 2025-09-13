package main

import (
	"TDMR87/go_protohackers/internal/server"
	"bufio"
	"net"
	"testing"
	"time"
)

func TestRegex(t *testing.T) {
	testCases := map[string]struct {
		input   string
		shouldMatch bool
	}{
		"test case 0": {
			input: tonysBogusCoinAddr,
			shouldMatch: true,
		},
		"test case 1": {
			input: "  7F1u3wSD5RbOHQmupo9nx4TsnQ  ",
			shouldMatch: true,
		},
		"test case 2": {
			input: "7wertyui9gJZe0LM0g1gdfgdfgZauHuiShI",
			shouldMatch: true,
		},
		"test case 3": {
			input: "7iKDZEwPZSqIvDnHvVN2r0hUWXD5rHX sdfdsf",
			shouldMatch: true,
		},
		"test case 4": {
			input: "weqwe234 7LOrwbDlS8NujgjddyogWgIM93MV5N2VR 42342",
			shouldMatch: true,
		},
		"test case 5": {
			input: "too short 7wertyui9JZe0LM0g1ZauHui",
			shouldMatch: false,
		},
		"test case 6": {
			input: "too long 7wertyui9JZe0LM0g1ZauHuifgfdgdg4dssf",
			shouldMatch: false,
		},
		"test case 7": {
			input: "7vi9jJ8u5Av1WxgLvdMDSzES8KsYc7-EoQVwDjwjb4X1cnx6YaVgCv2hiLykCQ-1234",
			shouldMatch: false,
		},
	}

	for name, tt := range testCases {
		isMatch, _ := bogusCoinRegex.MatchString(tt.input)
		if tt.shouldMatch != isMatch {
			t.Fatalf("%s : ShouldMatch '%v' but IsMatch '%v'", name, tt.shouldMatch, isMatch)
		}
	}
}

func TestBadNameClient(t *testing.T) {
	listener, err := server.StartTcpListener(":0", handle)
	if err != nil {
		t.Fatal("Error starting server:", err)
	}
	defer listener.Close()

	// Connect as User1
	conn1, _ := net.Dial("tcp", listener.Addr().String())
	scanner1 := bufio.NewScanner(conn1)
	if scanner1.Scan() {}
	conn1.Write([]byte("User1\n"))
	if scanner1.Scan() {}

	// Connect as User2
	conn2, _ := net.Dial("tcp", listener.Addr().String())
	scanner2 := bufio.NewScanner(conn2)
	if scanner2.Scan() {}

	// Disconnect User2 before sending newline delimiter
	conn2.Write([]byte("User2"))
	conn2.Close()

	// User1 should not see User2 join
	done := make(chan struct{})
	go func() {
		if scanner1.Scan() {
			line := scanner1.Text()
			if line == "* User2 has joined the room" {
				t.Errorf("User1 received unexpected message: %s", line)
			}
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
	}
}