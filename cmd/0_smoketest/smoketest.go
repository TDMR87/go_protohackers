package main

import (
	"TDMR87/go_protohackers/internal/server"
	"io"
	"log"
	"net"
)

func main() {
    server.StartTcpListener(":8080", handle)
	select {}
}

func handle(conn net.Conn) {
    defer conn.Close()

    buf := make([]byte, 1024)
    for {
        n, err := conn.Read(buf)
        if err != nil {
            if err == io.EOF {
				log.Println("Connection closed by client")
                return
            } else {
                log.Println("Read error:", err)
                return
            }
        }

        _, err = conn.Write(buf[:n])
        if err != nil {
            log.Println("Write error:", err)
            return
        }
    }
}