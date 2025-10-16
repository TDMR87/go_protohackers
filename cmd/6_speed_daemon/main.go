package main

import (
	"TDMR87/go_protohackers/internal/server"
	"fmt"
	"net"
	"time"
)

func main() {
	server.StartTcpListener(":8080", handle)
}

func handle(conn net.Conn) {
	defer conn.Close()

	reader := NewMessageReader(conn)

	for {
		msg, err := reader.NextMessage()
		if err != nil {
			return
		}

		switch m := msg.(type) {
		case WantHeartBeat:
			if _, exists := heartbeatClients[conn]; exists {
				response, _ := Error{Msg: "Heart beats are already being sent to this client"}.Encode()
				conn.Write(response)
				conn.Close()
				return
			}
			if m.Interval > 0 {
				go sendHeartBeat(conn, m.Interval)
			}
		case Plate:
			response, _ := Error{Msg: "Not implemented"}.Encode()
			conn.Write(response)
			conn.Close()
			return
		case IAmCamera:
			response, _ := Error{Msg: "Not implemented"}.Encode()
			conn.Write(response)
			conn.Close()
			return
		case IAmDispatcher:
			response, _ := Error{Msg: "Not implemented"}.Encode()
			conn.Write(response)
			conn.Close()
			return
		default:
			fmt.Printf("Unknown message type: %T\n", m)
		}
	}
}

var heartbeatClients = make(map[net.Conn]struct{})

func sendHeartBeat(conn net.Conn, deciSeconds uint32) {
	heartbeatClients[conn] = struct{}{}
	interval := time.Duration(deciSeconds*100) * time.Millisecond
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	defer delete(heartbeatClients, conn)

	for range ticker.C {
		_, err := conn.Write(HeartBeat{}.Encode())
		if err != nil {
			return
		}
	}
}