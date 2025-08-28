package main

import (
	"TDMR87/protohackers/server"
	"encoding/binary"
	"net"
	"testing"

	"github.com/google/uuid"
)

func TestInsertData(t *testing.T) {
	if len(sessionData) != 0 {
		t.Fatal("Data already exists")
	}
	handleInsert(makeMessage('I', 1100, 100), SessionId(uuid.New()))
	if len(sessionData) != 1 {
		t.Fatal("Inserting data failed")
	}
}

func TestQueryData(t *testing.T) {
	sessionId := SessionId(uuid.New())
	handleInsert(makeMessage('I', 1000, 100), sessionId)
	resultBytes := handleQuery(makeMessage('Q', 999, 1001), sessionId)
	resultVal := int32(binary.BigEndian.Uint32(resultBytes))
	if resultVal != 100 {
		t.Fatalf("Query failed. Expected %v, got %v", 100, resultVal)
	}
}

func TestCalculateMean(t *testing.T) {
	res := 7896543723626 / 61189
	var expectedValue = 129051687
	if res != expectedValue {
		t.Fatalf("Expected %v, got %v", res, expectedValue)
	}
}

func TestServer(t *testing.T) {
	testCases := map[string]struct {
		InsertMessages [][]byte
		QueryMessage   []byte
		ExpectedResult int32
		ExpectToFail bool
	}{
		"Invalid action": {
			InsertMessages: [][]byte{
				makeMessage('X', 12345, 100), // X is not an allowed value
			},
			QueryMessage:   nil,
			ExpectedResult: 0,
			ExpectToFail: true,
		},
		"No prices in requested period": {
			InsertMessages: [][]byte{},
			QueryMessage:   makeMessage('Q', 1, 2),
			ExpectedResult: 0,
		},
		"Mintime comes after maxtime": {
			InsertMessages: [][]byte{},
			QueryMessage:   makeMessage('Q', 2, 1),
			ExpectedResult: 0,
		},
		"Example": {
			InsertMessages: [][]byte{
				makeMessage('I', 12345, 101),
				makeMessage('I', 12346, 102),
				makeMessage('I', 12347, 100),
				makeMessage('I', 40960, 5),
			},
			QueryMessage:   makeMessage('Q', 12288, 16384),
			ExpectedResult: 101,
		},
		"Negative prices": {
			InsertMessages: [][]byte{
				makeMessage('I', 1001, 100),
				makeMessage('I', 1002, -50),
				makeMessage('I', 1003, 100),
			},
			QueryMessage:   makeMessage('Q', 1000, 1003),
			ExpectedResult: 50,
		},
		"Decimal result is rounded down": {
			InsertMessages: [][]byte{
				makeMessage('I', 9999, 97),   //
				makeMessage('I', 9998, 23),   // Actual mean is 373.67
				makeMessage('I', 9997, 1001), //
			},
			QueryMessage:   makeMessage('Q', 9995, 10_000),
			ExpectedResult: 373, // Server should round down the result
		},
		"Very large numbers": {
			InsertMessages: [][]byte{
				makeMessage('I', 1000000, 2117483647),
				makeMessage('I', 1000001, 2127483647),
				makeMessage('I', 1000002, 2107482647),
				makeMessage('I', 1000003, 2147433647),
				makeMessage('I', 1000004, 2142483647),
				makeMessage('I', 1000005, 2127483647),
				makeMessage('I', 1000006, 2146453647),
				makeMessage('I', 1000007, 2145481647),
				makeMessage('I', 1000008, 2143483647),
				makeMessage('I', 1000009, 2117483647),
			},
			QueryMessage:   makeMessage('Q', 999_999, 1_000_011),
			ExpectedResult: 2_132_275_347,
		},
	}

	listener, err := server.StartListener(":0", handle)
	if err != nil {
		t.Fatal("Error starting server:", err)
	}
	defer listener.Close()

	for name, tt := range testCases {
		t.Run(name, func(t *testing.T) {
			conn, err := net.Dial("tcp", listener.Addr().String())
			if err != nil {
				t.Fatal("Error connecting to server:", err)
			}
			defer conn.Close()

			// Insert data
			for _, msg := range tt.InsertMessages {
				conn.Write(msg)
			}
			
			if tt.ExpectToFail {
				buf := make([]byte, 1024)
				n, _ :=conn.Read(buf)
				errorMsg := string(buf[:n])
				if errorMsg != "malformed" {
					t.Fatalf("Test '%s' failed. Expected error response 'malformed', got '%s'", name, errorMsg)
				}
			}

			// Query data
			conn.Write(tt.QueryMessage)

			// Assert the result of the query
			buf := make([]byte, 8) // Result must be int32
			conn.Read(buf)
			result := int32(binary.BigEndian.Uint32(buf))
			if result != tt.ExpectedResult {
				t.Fatalf("Test '%s' failed. Invalid query result. Expected %v, got %v", name, tt.ExpectedResult, result)
			}
		})
	}
}

func makeMessage(firstByte byte, firstInt, secondInt int32) []byte {
	msg := make([]byte, 9)
	msg[0] = firstByte
	binary.BigEndian.PutUint32(msg[1:5], uint32(firstInt))
	binary.BigEndian.PutUint32(msg[5:9], uint32(secondInt))
	return msg
}