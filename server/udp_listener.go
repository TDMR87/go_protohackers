package server

import (
	"log"
	"net"
)

func StartUdpListener(addr string, handle func (*net.UDPConn, []byte, int, *net.UDPAddr, error)) (conn *net.UDPConn, err error) {
	urpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Println("Error starting server:", err)
		return nil, err
	}

	conn, err = net.ListenUDP("udp", urpAddr)
	if err != nil {
		log.Println("Error starting server:", err)
		return nil, err
	}

	go func () {
		for {
			buf := make([]byte, 1000)
			n, addr, err := conn.ReadFromUDP(buf)
			if n == len(buf) {
				continue // Messages must be under 1000 bytes
			}
			handle(conn, buf, n, addr, err)
		}
	}()

	log.Println("********************************")
	log.Println("Server is listening on", conn.LocalAddr().String())
	log.Println("********************************")

	return conn, nil
}