package main

import (
	"TDMR87/go_protohackers/internal/server"
	"net"
)

func main() {
	server.StartTcpListener(":8080", handle)
}

func handle(conn net.Conn) {

}