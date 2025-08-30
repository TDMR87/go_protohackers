package main

import (
	"TDMR87/protohackers/server"
	"bufio"
	"net"
	"testing"
)

func TestConnectToChatRoom(t *testing.T) {
	listener, err := server.StartListener(":0", handle)
	if err != nil {
		t.Fatal("Error starting server:", err)
	}
	defer listener.Close()

	conn, _ := net.Dial("tcp", listener.Addr().String())
	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		expectedPrompt := "Welcome to budgetchat! What shall I call you?"
		welcomePrompt := scanner.Text()
		if welcomePrompt != expectedPrompt {
			t.Fatalf("Expected '%s', got '%s'", expectedPrompt, welcomePrompt)
		}
	}

	conn.Write(NewChatMessage("Manuel\n"))

	if scanner.Scan() {
		// No other users joined, so the list must be empty
		expectedWelcomeMessage := "* The room contains: "
		welcomeMessage := scanner.Text()
		if welcomeMessage != expectedWelcomeMessage {
			t.Fatalf("Expected '%s', got '%s'", expectedWelcomeMessage, welcomeMessage)
		}
	}
}

func TestTooLongUsername(t *testing.T) {
	listener, err := server.StartListener(":0", handle)
	if err != nil {
		t.Fatal("Error starting server:", err)
	}
	defer listener.Close()

	conn, _ := net.Dial("tcp", listener.Addr().String())
	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		_ = scanner.Text() // Reads the username prompt
	}

	conn.Write(NewChatMessage("thisIsATooLongUsername\n"))

	if scanner.Scan() {
		errMsg := scanner.Text()
		expectedErrMsg := "Invalid username. Usernames must be 1-16 characters long and must consist entirely of alphanumeric characters (uppercase, lowercase, and digits)"
		if errMsg != expectedErrMsg {
			t.Fatalf("Expected '%s', got '%s'", expectedErrMsg, errMsg)
		}
	}
}

func TestTooShortUsername(t *testing.T) {
	listener, err := server.StartListener(":0", handle)
	if err != nil {
		t.Fatal("Error starting server:", err)
	}
	defer listener.Close()

	conn, _ := net.Dial("tcp", listener.Addr().String())
	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		_ = scanner.Text() // Reads the username prompt
	}

	// Sends a too short name
	conn.Write(NewChatMessage("\n"))

	if scanner.Scan() {
		errMsg := scanner.Text()
		expectedErrMsg := "Invalid username. Usernames must be 1-16 characters long and must consist entirely of alphanumeric characters (uppercase, lowercase, and digits)"
		if errMsg != expectedErrMsg {
			t.Fatalf("Expected '%s', got '%s'", expectedErrMsg, errMsg)
		}
	}
}

func TestAsciiUsername(t *testing.T) {
	listener, err := server.StartListener(":0", handle)
	if err != nil {
		t.Fatal("Error starting server:", err)
	}
	defer listener.Close()

	conn, _ := net.Dial("tcp", listener.Addr().String())
	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		_ = scanner.Text() // Reads the username prompt
	}

	// Sends a username with prohibited characters
	conn.Write(ChatMessage("<<special username>>\n"))

	if scanner.Scan() {
		errMsg := scanner.Text()
		expectedErrMsg := "Invalid username. Usernames must be 1-16 characters long and must consist entirely of alphanumeric characters (uppercase, lowercase, and digits)"
		if errMsg != expectedErrMsg {
			t.Fatalf("Expected '%s', got '%s'", expectedErrMsg, errMsg)
		}
	}
}
