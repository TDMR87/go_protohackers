package main

import (
	"TDMR87/go_protohackers/internal/server"
	"fmt"
	"net"
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
		case Error:
			fmt.Println("Got Error message:", m.Msg)
		case Plate:
			fmt.Println("Got Plate:", m.Plate, "Timestamp:", m.Timestamp)
		case Ticket:
			fmt.Println("Got Ticket:", m.Plate, "Speed:", m.Speed)
		default:
			fmt.Printf("Unknown message type: %T\n", m)
		}
	}
}
