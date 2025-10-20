package main

import (
	"TDMR87/go_protohackers/internal/server"
	"net"
	"testing"
	"time"
)

func resetGlobalState() {
	heartbeatClients = make(map[net.Conn]struct{})
	dispatchers = make(map[net.Conn]IAmDispatcher)
	cameraClients = make(map[net.Conn]IAmCamera)
	cameraPlateSnapshots = make(map[IAmCamera]Plate)
	sentTickets = make(map[string][]uint32)
	outgoingTickets = make([]Ticket, 0)
}

func Test_WantHeartBeat_OnlyOnePerClientAllowed(t *testing.T) {
	resetGlobalState()
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
	resetGlobalState()
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
	resetGlobalState()
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
	resetGlobalState()
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
	resetGlobalState()
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
	resetGlobalState()
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
	if response.Msg != "Client must be identified as a camera to send a plate" {
		t.Fatalf("expected error message 'Client must be identified as a camera to send a plate', got '%s'", response.Msg)
	}
}

func Test_SendTicket(t *testing.T) {
	resetGlobalState()
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
		t.Fatal("Expected speeding ticket to be outgoing, but it was not")
	}

	// Now a dispatcher connects to receive the ticket
	dispatcherConn, _ := net.Dial("tcp", listener.Addr().String())
	defer dispatcherConn.Close()
	dispatcher := IAmDispatcher{Numroads: 1, Roads: []uint16{123}}
	dispatcherConn.Write(dispatcher.Encode())
	time.Sleep(100 * time.Millisecond)

	if len(outgoingTickets) != 0 {
		t.Fatal("Expected outgoing ticket to have been sent, but it was not sent")
	}
}

func Test_IAmDispatcher_OnlyOnePerClientAllowed(t *testing.T) {
	resetGlobalState()
	listener, err := server.StartTcpListener(":0", handle)
	if err != nil {
		t.Fatal("Error starting server:", err)
	}
	defer listener.Close()

	conn, _ := net.Dial("tcp", listener.Addr().String())
	defer conn.Close()

	// Send first IAmDispatcher message
	firstMsg := IAmDispatcher{Numroads: 2, Roads: []uint16{123, 456}}
	conn.Write(firstMsg.Encode())
	time.Sleep(100 * time.Millisecond)

	// Send second IAmDispatcher message
	secondMsg := IAmDispatcher{Numroads: 1, Roads: []uint16{456}}
	conn.Write(secondMsg.Encode())
	time.Sleep(100 * time.Millisecond)

	buf := make([]byte, Error{}.Size())
	n, _ := conn.Read(buf)
	response, _ := Error{}.Decode(buf[:n])
	if response.Msg != "Client is already identified as a dispatcher" {
		t.Fatalf("expected error message 'Client is already identified as a dispatcher', got '%s'", response.Msg)
	}
}

func Test_CompleteScenario_MultipleCamerasDispatchersAndPlates(t *testing.T) {
	resetGlobalState()
	listener, err := server.StartTcpListener(":0", handle)
	if err != nil {
		t.Fatal("Error starting server:", err)
	}
	defer listener.Close()

	// Camera 1 on Road 66, Mile 100, Limit 60 mph
	camera1Conn, _ := net.Dial("tcp", listener.Addr().String())
	defer camera1Conn.Close()
	camera1 := IAmCamera{Road: 66, Mile: 100, Limit: 60}
	camera1Conn.Write(camera1.Encode())
	time.Sleep(50 * time.Millisecond)

	// Camera 2 on Road 66, Mile 110, Limit 60 mph
	camera2Conn, _ := net.Dial("tcp", listener.Addr().String())
	defer camera2Conn.Close()
	camera2 := IAmCamera{Road: 66, Mile: 110, Limit: 60}
	camera2Conn.Write(camera2.Encode())
	time.Sleep(50 * time.Millisecond)

	// Camera 3 on Road 123, Mile 50, Limit 70 mph
	camera3Conn, _ := net.Dial("tcp", listener.Addr().String())
	defer camera3Conn.Close()
	camera3 := IAmCamera{Road: 123, Mile: 50, Limit: 70}
	camera3Conn.Write(camera3.Encode())
	time.Sleep(50 * time.Millisecond)

	// Camera 4 on Road 123, Mile 60, Limit 70 mph
	camera4Conn, _ := net.Dial("tcp", listener.Addr().String())
	defer camera4Conn.Close()
	camera4 := IAmCamera{Road: 123, Mile: 60, Limit: 70}
	camera4Conn.Write(camera4.Encode())
	time.Sleep(50 * time.Millisecond)

	// Car "SPEEDY1" on Road 66 - speeding (100 miles in 1 hour = 100 mph, limit is 60)
	plate1, _ := Plate{Plate: "SPEEDY1", Timestamp: 0}.Encode()
	camera1Conn.Write(plate1)
	time.Sleep(50 * time.Millisecond)

	plate2, _ := Plate{Plate: "SPEEDY1", Timestamp: 360}.Encode() // 1 hour later, 10 miles ahead
	camera2Conn.Write(plate2)
	time.Sleep(50 * time.Millisecond)

	// Car "LEGAL1" on Road 66 - not speeding (10 miles in 2 hours = 5 mph)
	plate3, _ := Plate{Plate: "LEGAL1", Timestamp: 0}.Encode()
	camera1Conn.Write(plate3)
	time.Sleep(50 * time.Millisecond)

	plate4, _ := Plate{Plate: "LEGAL1", Timestamp: 7200}.Encode() // 2 hours later
	camera2Conn.Write(plate4)
	time.Sleep(50 * time.Millisecond)

	// Car "SPEEDY2" on Road 123 - speeding (10 miles in 0.1 hour = 100 mph, limit is 70)
	plate5, _ := Plate{Plate: "SPEEDY2", Timestamp: 0}.Encode()
	camera3Conn.Write(plate5)
	time.Sleep(50 * time.Millisecond)

	plate6, _ := Plate{Plate: "SPEEDY2", Timestamp: 360}.Encode() // 0.1 hour later
	camera4Conn.Write(plate6)
	time.Sleep(50 * time.Millisecond)

	// Car "LEGAL2" on Road 123 - not speeding
	plate7, _ := Plate{Plate: "LEGAL2", Timestamp: 0}.Encode()
	camera3Conn.Write(plate7)
	time.Sleep(50 * time.Millisecond)

	plate8, _ := Plate{Plate: "LEGAL2", Timestamp: 600}.Encode()
	camera4Conn.Write(plate8)
	time.Sleep(100 * time.Millisecond)

	// Check that we have 2 outgoing tickets (SPEEDY1 and SPEEDY2)
	if len(outgoingTickets) != 2 {
		t.Fatalf("Expected 2 outgoing tickets, got %d. Tickets: %v", len(outgoingTickets), outgoingTickets)
	}

	// Dispatcher 1 handles Road 66
	dispatcher1Conn, _ := net.Dial("tcp", listener.Addr().String())
	defer dispatcher1Conn.Close()
	dispatcher1Conn.Write(IAmDispatcher{Numroads: 1, Roads: []uint16{66}}.Encode())
	time.Sleep(100 * time.Millisecond)

	// Dispatcher 1 should receive ticket for SPEEDY1
	buf := make([]byte, 1024)
	dispatcher1Conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	n, err := dispatcher1Conn.Read(buf)
	if err != nil {
		t.Fatal("Expected to receive ticket for SPEEDY1:", err)
	}

	ticket, err := Ticket{}.Decode(buf[:n])
	if err != nil {
		t.Fatal("Error decoding ticket:", err)
	}

	if ticket.Plate != "SPEEDY1" {
		t.Fatalf("Expected ticket for SPEEDY1, got %s", ticket.Plate)
	}
	if ticket.Road != 66 {
		t.Fatalf("Expected ticket for Road 66, got %d", ticket.Road)
	}

	// Now only 1 ticket should remain (SPEEDY2)
	time.Sleep(100 * time.Millisecond)
	if len(outgoingTickets) != 1 {
		t.Fatalf("Expected 1 remaining ticket, got %d", len(outgoingTickets))
	}

	// Dispatcher 2 handles multiple roads including Road 123
	dispatcher2Conn, _ := net.Dial("tcp", listener.Addr().String())
	defer dispatcher2Conn.Close()
	dispatcher2 := IAmDispatcher{Numroads: 3, Roads: []uint16{99, 123, 456}}
	dispatcher2Conn.Write(dispatcher2.Encode())
	time.Sleep(100 * time.Millisecond)

	// Dispatcher 2 should receive ticket for SPEEDY2
	dispatcher2Conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	n, err = dispatcher2Conn.Read(buf)
	if err != nil {
		t.Fatal("Expected to receive ticket for SPEEDY2:", err)
	}

	ticket, err = Ticket{}.Decode(buf[:n])
	if err != nil {
		t.Fatal("Error decoding ticket:", err)
	}

	if ticket.Plate != "SPEEDY2" {
		t.Fatalf("Expected ticket for SPEEDY2, got %s", ticket.Plate)
	}
	if ticket.Road != 123 {
		t.Fatalf("Expected ticket for Road 123, got %d", ticket.Road)
	}

	// All tickets should now be sent
	time.Sleep(100 * time.Millisecond)
	if len(outgoingTickets) != 0 {
		t.Fatalf("Expected all tickets to be sent, got %d remaining", len(outgoingTickets))
	}

	// Test heartbeats alongside everything else
	heartbeatConn, _ := net.Dial("tcp", listener.Addr().String())
	defer heartbeatConn.Close()
	heartbeatConn.Write(WantHeartBeat{Interval: 1}.Encode())

	heartbeatBuf := make([]byte, HeartBeat{}.Size())
	heartbeatConn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	n, err = heartbeatConn.Read(heartbeatBuf)
	if err != nil {
		t.Fatal("Expected heartbeat:", err)
	}

	_, err = HeartBeat{}.Decode(heartbeatBuf[:n])
	if err != nil {
		t.Fatal("Error decoding heartbeat:", err)
	}
}

func Test_SingleCar_DispatcherConnectsAfterSpeeding(t *testing.T) {
	resetGlobalState()
	listener, err := server.StartTcpListener(":0", handle)
	if err != nil {
		t.Fatal("Error starting server:", err)
	}
	defer listener.Close()

	// Camera 1 on Road 66, Mile 8, Limit 60 mph
	camera1Conn, _ := net.Dial("tcp", listener.Addr().String())
	defer camera1Conn.Close()
	camera1 := IAmCamera{Road: 66, Mile: 8, Limit: 60}
	camera1Conn.Write(camera1.Encode())
	time.Sleep(50 * time.Millisecond)

	// Camera 2 on Road 66, Mile 9, Limit 60 mph
	camera2Conn, _ := net.Dial("tcp", listener.Addr().String())
	defer camera2Conn.Close()
	camera2 := IAmCamera{Road: 66, Mile: 9, Limit: 60}
	camera2Conn.Write(camera2.Encode())
	time.Sleep(50 * time.Millisecond)

	// Car passes camera 1 at timestamp 0
	plate1, _ := Plate{Plate: "UN1X", Timestamp: 0}.Encode()
	camera1Conn.Write(plate1)
	time.Sleep(50 * time.Millisecond)

	// Car passes camera 2 at timestamp 45 (45 seconds later)
	// Distance: 1 mile in 45 seconds = 80 mph (exceeds 60 mph limit)
	plate2, _ := Plate{Plate: "UN1X", Timestamp: 45}.Encode()
	camera2Conn.Write(plate2)
	time.Sleep(100 * time.Millisecond)

	// Verify ticket was created
	outgoingTicketsMu.Lock()
	ticketCount := len(outgoingTickets)
	outgoingTicketsMu.Unlock()

	if ticketCount != 1 {
		t.Fatalf("Expected 1 ticket to be created, got %d", ticketCount)
	}

	// NOW dispatcher connects (after speeding already occurred)
	dispatcherConn, _ := net.Dial("tcp", listener.Addr().String())
	defer dispatcherConn.Close()
	dispatcher := IAmDispatcher{Numroads: 1, Roads: []uint16{66}}
	dispatcherConn.Write(dispatcher.Encode())

	// Dispatcher should receive the ticket within reasonable time
	buf := make([]byte, 1024)
	dispatcherConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := dispatcherConn.Read(buf)

	if err != nil {
		t.Fatal("Expected to receive speeding ticket, but got error:", err)
	}

	ticket, err := Ticket{}.Decode(buf[:n])
	if err != nil {
		t.Fatal("Error decoding ticket:", err)
	}

	if ticket.Plate != "UN1X" {
		t.Fatalf("Expected ticket for UN1X, got %s", ticket.Plate)
	}
	if ticket.Road != 66 {
		t.Fatalf("Expected ticket for Road 66, got %d", ticket.Road)
	}
	if ticket.Mile1 != 8 {
		t.Fatalf("Expected Mile1 to be 8, got %d", ticket.Mile1)
	}
	if ticket.Mile2 != 9 {
		t.Fatalf("Expected Mile2 to be 9, got %d", ticket.Mile2)
	}
	if ticket.Timestamp1 != 0 {
		t.Fatalf("Expected Timestamp1 to be 0, got %d", ticket.Timestamp1)
	}
	if ticket.Timestamp2 != 45 {
		t.Fatalf("Expected Timestamp2 to be 45, got %d", ticket.Timestamp2)
	}

	// Speed should be 80 mph = 8000 (in hundredths)
	expectedSpeed := uint16(8000)
	if ticket.Speed != expectedSpeed {
		t.Fatalf("Expected speed to be %d, got %d", expectedSpeed, ticket.Speed)
	}

	// Verify ticket was removed from queue
	time.Sleep(100 * time.Millisecond)
	outgoingTicketsMu.Lock()
	remainingTickets := len(outgoingTickets)
	outgoingTicketsMu.Unlock()

	if remainingTickets != 0 {
		t.Fatalf("Expected 0 remaining tickets, got %d", remainingTickets)
	}
}

func Test_SingleCar_ObservationsInReverseOrder(t *testing.T) {
	resetGlobalState()
	listener, err := server.StartTcpListener(":0", handle)
	if err != nil {
		t.Fatal("Error starting server:", err)
	}
	defer listener.Close()

	// Camera 1 on Road 66, Mile 8, Limit 60 mph
	camera1Conn, _ := net.Dial("tcp", listener.Addr().String())
	defer camera1Conn.Close()
	camera1 := IAmCamera{Road: 66, Mile: 8, Limit: 60}
	camera1Conn.Write(camera1.Encode())
	time.Sleep(50 * time.Millisecond)

	// Camera 2 on Road 66, Mile 9, Limit 60 mph
	camera2Conn, _ := net.Dial("tcp", listener.Addr().String())
	defer camera2Conn.Close()
	camera2 := IAmCamera{Road: 66, Mile: 9, Limit: 60}
	camera2Conn.Write(camera2.Encode())
	time.Sleep(50 * time.Millisecond)

	// Dispatcher connects first
	dispatcherConn, _ := net.Dial("tcp", listener.Addr().String())
	defer dispatcherConn.Close()
	dispatcher := IAmDispatcher{Numroads: 1, Roads: []uint16{66}}
	dispatcherConn.Write(dispatcher.Encode())
	time.Sleep(50 * time.Millisecond)

	// Car passes camera 2 at timestamp 45
	plate2, _ := Plate{Plate: "UN1X", Timestamp: 45}.Encode()
	camera2Conn.Write(plate2)
	time.Sleep(50 * time.Millisecond)

	// Car passes camera 1 at timestamp 0 (earlier in time, but reported later)
	// This simulates out-of-order reporting
	plate1, _ := Plate{Plate: "UN1X", Timestamp: 0}.Encode()
	camera1Conn.Write(plate1)
	time.Sleep(100 * time.Millisecond)

	// Should still detect speeding (1 mile in 45 seconds = 80 mph)
	buf := make([]byte, 1024)
	dispatcherConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := dispatcherConn.Read(buf)

	if err != nil {
		t.Fatal("Expected to receive speeding ticket with reversed observations, but got error:", err)
	}

	ticket, err := Ticket{}.Decode(buf[:n])
	if err != nil {
		t.Fatal("Error decoding ticket:", err)
	}

	if ticket.Plate != "UN1X" {
		t.Fatalf("Expected ticket for UN1X, got %s", ticket.Plate)
	}

	// Ticket should have correct order (mile1 < mile2, timestamp1 < timestamp2)
	if ticket.Mile1 != 8 || ticket.Mile2 != 9 {
		t.Fatalf("Expected Mile1=8, Mile2=9, got Mile1=%d, Mile2=%d", ticket.Mile1, ticket.Mile2)
	}
	if ticket.Timestamp1 != 0 || ticket.Timestamp2 != 45 {
		t.Fatalf("Expected Timestamp1=0, Timestamp2=45, got Timestamp1=%d, Timestamp2=%d", ticket.Timestamp1, ticket.Timestamp2)
	}
}

func Test_PreventDuplicateTicketsOnSameDay(t *testing.T) {
	resetGlobalState()
	listener, err := server.StartTcpListener(":0", handle)
	if err != nil {
		t.Fatal("Error starting server:", err)
	}
	defer listener.Close()

	// Camera 1 on Road 66, Mile 8, Limit 60 mph
	camera1Conn, _ := net.Dial("tcp", listener.Addr().String())
	defer camera1Conn.Close()
	camera1 := IAmCamera{Road: 66, Mile: 8, Limit: 60}
	camera1Conn.Write(camera1.Encode())
	time.Sleep(50 * time.Millisecond)

	// Camera 2 on Road 66, Mile 9, Limit 60 mph
	camera2Conn, _ := net.Dial("tcp", listener.Addr().String())
	defer camera2Conn.Close()
	camera2 := IAmCamera{Road: 66, Mile: 9, Limit: 60}
	camera2Conn.Write(camera2.Encode())
	time.Sleep(50 * time.Millisecond)

	// Dispatcher connects
	dispatcherConn, _ := net.Dial("tcp", listener.Addr().String())
	defer dispatcherConn.Close()
	dispatcher := IAmDispatcher{Numroads: 1, Roads: []uint16{66}}
	dispatcherConn.Write(dispatcher.Encode())
	time.Sleep(50 * time.Millisecond)

	// First speeding violation
	// Speed: 1 mile in 45 seconds = 80 mph
	plate1a, _ := Plate{Plate: "UN1X", Timestamp: 0}.Encode()
	camera1Conn.Write(plate1a)
	time.Sleep(50 * time.Millisecond)

	plate1b, _ := Plate{Plate: "UN1X", Timestamp: 45}.Encode()
	camera2Conn.Write(plate1b)
	time.Sleep(100 * time.Millisecond)

	// Should receive first ticket
	buf := make([]byte, 1024)
	dispatcherConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	n, err := dispatcherConn.Read(buf)
	if err != nil {
		t.Fatal("Expected to receive first ticket:", err)
	}

	ticket1, err := Ticket{}.Decode(buf[:n])
	if err != nil {
		t.Fatal("Error decoding first ticket:", err)
	}
	if ticket1.Plate != "UN1X" {
		t.Fatalf("Expected ticket for UN1X, got %s", ticket1.Plate)
	}

	// Second speeding violation - SAME DAY (timestamp 1000)
	// Speed: 1 mile in 45 seconds = 80 mph
	plate2a, _ := Plate{Plate: "UN1X", Timestamp: 1000}.Encode()
	camera1Conn.Write(plate2a)
	time.Sleep(50 * time.Millisecond)

	plate2b, _ := Plate{Plate: "UN1X", Timestamp: 1045}.Encode()
	camera2Conn.Write(plate2b)
	time.Sleep(200 * time.Millisecond)

	// Should NOT receive a second ticket because it's the same day
	dispatcherConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	n, err = dispatcherConn.Read(buf)
	if err == nil {
		ticket2, _ := Ticket{}.Decode(buf[:n])
		t.Fatalf("Should not have received second ticket on the same day, but got ticket: %+v", ticket2)
	}

	// Third speeding violation - DIFFERENT DAY (timestamp 86400 = day 1)
	// Speed: 1 mile in 45 seconds = 80 mph
	plate3a, _ := Plate{Plate: "UN1X", Timestamp: 86400}.Encode()
	camera1Conn.Write(plate3a)
	time.Sleep(50 * time.Millisecond)

	plate3b, _ := Plate{Plate: "UN1X", Timestamp: 86445}.Encode()
	camera2Conn.Write(plate3b)
	time.Sleep(200 * time.Millisecond)

	// Should receive third ticket (different day)
	dispatcherConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	n, err = dispatcherConn.Read(buf)
	if err != nil {
		t.Fatal("Expected to receive third ticket (different day):", err)
	}

	ticket3, err := Ticket{}.Decode(buf[:n])
	if err != nil {
		t.Fatal("Error decoding third ticket:", err)
	}
	if ticket3.Plate != "UN1X" {
		t.Fatalf("Expected ticket for UN1X, got %s", ticket3.Plate)
	}
	if ticket3.Timestamp1 != 86400 {
		t.Fatalf("Expected ticket from day 1, got timestamp %d", ticket3.Timestamp1)
	}
}
