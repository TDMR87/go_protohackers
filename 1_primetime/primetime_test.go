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
			"prime number": {
				request:  `{"method":"isPrime","number":7}`,
				response: `{"method":"isPrime","prime":true}`,
			},
			"non-prime number": {
				request:  `{"method":"isPrime","number":6}`,
				response: `{"method":"isPrime","prime":false}`,
			},
			"zero number": {
				request:  `{"method":"isPrime","number":0}`,
				response: `{"method":"isPrime","prime":false}`,
			},
			"negative number": {
				request:  `{"method":"isPrime","number":-999}`,
				response: `{"method":"isPrime","prime":false}`,
			},
			"number as string": {
				request:  `{"method":"isPrime","number":"10"}`,
				response: `malformed`,
			},
			"missing number property": {
				request:  `{"method":"isPrime"}`,
				response: `malformed`,
			},
			"missing method property": {
				request:  `{"number":10}`,
				response: `malformed`,
			},
			"wrong method": {
				request:  `{"method":"isEven","number":10}`,
				response: `malformed`,
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
				t.Fatal("Error writing request to server:", err)
			}

			scanner := bufio.NewScanner(conn)
			if !scanner.Scan() {
				t.Fatal("No response from server")
			}
			if scanner.Text() != tt.response {
				t.Errorf("Expected response %q, got %q", tt.response, scanner.Text())
			}	
		}
	}