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

	if response.Msg != "Client is already receiving heartbeats" {
		t.Fatalf("expected error message 'Client is already receiving heartbeats', got '%s'", response.Msg)
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

func Test_IAmCamera_RegistersSuccessfully(t *testing.T) {
	listener, err := server.StartTcpListener(":0", handle)
	if err != nil {
		t.Fatal("Error starting server:", err)
	}
	defer listener.Close()

	conn, _ := net.Dial("tcp", listener.Addr().String())
	defer conn.Close()

	// Send IAmCamera message
	expectedCamera := IAmCamera{Road: 123, Mile: 8, Limit: 60}
	_, err = conn.Write(expectedCamera.Encode())

	if err != nil {
		t.Fatalf("Expected no error for valid IAmCamera, but got one: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Check that the camera was registered
	cameraRegistered := false
	for _, camera := range cameraClients {
		if camera == expectedCamera {
			cameraRegistered = true
			break
		}
	}

	if !cameraRegistered {
		t.Fatal("Camera was not registered")
	}
}

func Test_IAmCamera_OnlyOnePerClientAllowed(t *testing.T) {
	listener, err := server.StartTcpListener(":0", handle)
	if err != nil {
		t.Fatal("Error starting server:", err)
	}
	defer listener.Close()

	conn, _ := net.Dial("tcp", listener.Addr().String())
	defer conn.Close()

	// Send first IAmCamera message
	firstMsg := IAmCamera{Road: 123, Mile: 8, Limit: 60}
	conn.Write(firstMsg.Encode())
	time.Sleep(100 * time.Millisecond)

	// Send second IAmCamera message
	secondMsg := IAmCamera{Road: 456, Mile: 10, Limit: 70}
	conn.Write(secondMsg.Encode())

	buf := make([]byte, Error{}.Size())
	n, _ := conn.Read(buf)
	response, _ := Error{}.Decode(buf[:n])
	if response.Msg != "Client is already identified as a camera" {
		t.Fatalf("expected error message 'Client is already identified as a camera', got '%s'", response.Msg)
	}
}

func Test_ClientMustBeACamera_ToSendPlate(t *testing.T) {
	listener, err := server.StartTcpListener(":0", handle)
	if err != nil {
		t.Fatal("Error starting server:", err)
	}
	defer listener.Close()

	conn, _ := net.Dial("tcp", listener.Addr().String())
	defer conn.Close()

	plate, err := Plate{Plate: "ABCD1234", Timestamp: 123456}.Encode()
	if err != nil {
		t.Fatal("Error encoding plate:", err)
	}

	// Send Plate message without identifying as a camera
	conn.Write(plate)
	time.Sleep(100 * time.Millisecond)

	buf := make([]byte, Error{}.Size())
	n, _ := conn.Read(buf)
	response, _ := Error{}.Decode(buf[:n])
	if response.Msg != "Client sent a plate but is not identified as a camera" {
		t.Fatalf("expected error message 'Client sent a plate but is not identified as a camera', got '%s'", response.Msg)
	}
}

func Test_SendTicket(t *testing.T) {
	listener, err := server.StartTcpListener(":0", handle)
	if err != nil {
		t.Fatal("Error starting server:", err)
	}
	defer listener.Close()

	// First camera connects and sends a plate
	cameraOneConn, _ := net.Dial("tcp", listener.Addr().String())
	defer cameraOneConn.Close()
	cameraOne := IAmCamera{Road: 123, Mile: 8, Limit: 60}
	cameraOneConn.Write(cameraOne.Encode())
	time.Sleep(100 * time.Millisecond)
	plate, _ := Plate{Plate: "ABCD1234", Timestamp: 0}.Encode()
	cameraOneConn.Write(plate)
	time.Sleep(100 * time.Millisecond)

	// Second camera connects and sends a plate
	cameraTwoConn, _ := net.Dial("tcp", listener.Addr().String())
	defer cameraTwoConn.Close()	
	cameraTwo := IAmCamera{Road: 123, Mile: 9, Limit: 60}
	cameraTwoConn.Write(cameraTwo.Encode())
	time.Sleep(100 * time.Millisecond)
	plate, _ = Plate{Plate: "ABCD1234", Timestamp: 45}.Encode()
	cameraTwoConn.Write(plate)
	time.Sleep(100 * time.Millisecond)

	if len(outgoingTickets) != 1 {
		t.Fatal("Expected speeding ticket to be sent, but it was not sent")
	}
}
