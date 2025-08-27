package server

import (
	"log"
	"net"
)

func StartListener(addr string, handle func(net.Conn)) (net.Listener, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Println("Error starting server:", err)
		return nil, err
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Println("Error accepting connection:", err)
				continue
			}

			go handle(conn)
		}
	}()

	log.Println("********************************")
	log.Println("Server is listening on", listener.Addr().String())
	log.Println("********************************")
	return listener, nil
}