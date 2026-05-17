package main

import (
	"TDMR87/go_protohackers/internal/server"
	"net"
	"sync"
	"time"
)

type Server struct {
	mu                   sync.Mutex
	heartbeatClients     map[net.Conn]struct{}
	dispatchers          map[net.Conn]IAmDispatcher
	cameraClients        map[net.Conn]IAmCamera
	cameraPlateSnapshots map[Plate]IAmCamera
	sentTickets          map[string][]uint32
	outgoingTickets      []Ticket
}

func NewServer() *Server {
	return &Server{
		heartbeatClients:     make(map[net.Conn]struct{}),
		dispatchers:          make(map[net.Conn]IAmDispatcher),
		cameraClients:        make(map[net.Conn]IAmCamera),
		cameraPlateSnapshots: make(map[Plate]IAmCamera),
		sentTickets:          make(map[string][]uint32),
	}
}

func main() {
	s := NewServer()
	server.StartTcpListener(":8080", s.handle)
	select {}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	defer func() {
		s.mu.Lock()
		delete(s.heartbeatClients, conn)
		delete(s.cameraClients, conn)
		delete(s.dispatchers, conn)
		s.mu.Unlock()
	}()

	reader := NewMessageReader(conn)

	for {
		message, err := reader.NextMessage()
		if err != nil {
			response, _ := Error{Msg: err.Error()}.Encode()
			conn.Write(response)
			return
		}

		s.mu.Lock()

		switch msg := message.(type) {
		case WantHeartBeat:
			_, exists := s.heartbeatClients[conn]
			if exists {
				s.mu.Unlock()
				response, _ := Error{Msg: "Client is already receiving heartbeats"}.Encode()
				conn.Write(response)
				return
			}
			if msg.Interval > 0 {
				go s.sendHeartBeat(conn, msg.Interval)
			}

		case IAmCamera:
			_, exists := s.cameraClients[conn]
			if exists {
				s.mu.Unlock()
				response, _ := Error{Msg: "Client is already identified as a camera"}.Encode()
				conn.Write(response)
				return
			}
			s.cameraClients[conn] = msg

		case Plate:
			camera, exists := s.cameraClients[conn]
			if !exists {
				s.mu.Unlock()
				response, _ := Error{Msg: "Client must be identified as a camera to send a plate"}.Encode()
				conn.Write(response)
				continue
			}
			s.cameraPlateSnapshots[msg] = camera
			s.handlePlate(msg, camera)
			s.sendTickets()

		case IAmDispatcher:
			_, exists := s.dispatchers[conn]
			if exists {
				s.mu.Unlock()
				response, _ := Error{Msg: "Client is already identified as a dispatcher"}.Encode()
				conn.Write(response)
				return
			}
			s.dispatchers[conn] = msg
			s.sendTickets()

		default:
			s.mu.Unlock()
			response, _ := Error{Msg: "Unknown message received from MessageReader"}.Encode()
			conn.Write(response)
			continue
		}

		s.mu.Unlock()
	}
}

func (s *Server) sendHeartBeat(conn net.Conn, deciSeconds uint32) {
	s.mu.Lock()
	s.heartbeatClients[conn] = struct{}{}
	s.mu.Unlock()

	interval := time.Duration(deciSeconds*100) * time.Millisecond
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	defer func() {
		s.mu.Lock()
		delete(s.heartbeatClients, conn)
		s.mu.Unlock()
	}()

	for range ticker.C {
		_, err := conn.Write(HeartBeat{}.Encode())
		if err != nil {
			return
		}
	}
}

func (s *Server) handlePlate(currentPlate Plate, currentCamera IAmCamera) {
outerloop:
	for previousCameraPlate, previousCamera := range s.cameraPlateSnapshots {
		if previousCamera == currentCamera ||
			previousCameraPlate.Plate != currentPlate.Plate ||
			previousCamera.Road != currentCamera.Road {
			continue
		}

		currentDay := currentPlate.Timestamp / 86400
		previousDay := previousCameraPlate.Timestamp / 86400

		if _, exists := s.sentTickets[currentPlate.Plate]; exists {
			for _, day := range s.sentTickets[currentPlate.Plate] {
				if day >= previousDay && day <= currentDay {
					continue outerloop
				}
			}
		}

		var distanceDiff uint16
		var timeDiff uint32
		if currentCamera.Mile > previousCamera.Mile {
			distanceDiff = currentCamera.Mile - previousCamera.Mile
			timeDiff = currentPlate.Timestamp - previousCameraPlate.Timestamp
		} else {
			distanceDiff = previousCamera.Mile - currentCamera.Mile
			timeDiff = previousCameraPlate.Timestamp - currentPlate.Timestamp
		}

		if timeDiff == 0 {
			continue
		}

		speedInMph := (float64(distanceDiff) / float64(timeDiff)) * 3600.0
		if speedInMph < float64(currentCamera.Limit) {
			continue
		}

		for day := previousDay; day <= currentDay; day++ {
			s.sentTickets[currentPlate.Plate] = append(s.sentTickets[currentPlate.Plate], day)
		}

		if currentCamera.Mile > previousCamera.Mile {
			s.outgoingTickets = append(s.outgoingTickets, Ticket{
				Plate:      currentPlate.Plate,
				Road:       currentCamera.Road,
				Mile1:      previousCamera.Mile,
				Timestamp1: previousCameraPlate.Timestamp,
				Mile2:      currentCamera.Mile,
				Timestamp2: currentPlate.Timestamp,
				Speed:      uint16(speedInMph) * 100,
			})
		} else {
			s.outgoingTickets = append(s.outgoingTickets, Ticket{
				Plate:      currentPlate.Plate,
				Road:       currentCamera.Road,
				Mile1:      currentCamera.Mile,
				Timestamp1: currentPlate.Timestamp,
				Mile2:      previousCamera.Mile,
				Timestamp2: previousCameraPlate.Timestamp,
				Speed:      uint16(speedInMph) * 100,
			})
		}

		s.sendTickets()
		break
	}
}

func (s *Server) sendTickets() {
	ticketsCopy := make([]Ticket, len(s.outgoingTickets))
	copy(ticketsCopy, s.outgoingTickets)

ticketLoop:
	for _, ticket := range ticketsCopy {
		for dispatcherConn, dispatcher := range s.dispatchers {
			for _, dispatcherRoad := range dispatcher.Roads {
				if dispatcherRoad == ticket.Road {
					ticketBytes, err := ticket.Encode()
					if err != nil {
						response, _ := Error{Msg: "Failed to encode ticket"}.Encode()
						dispatcherConn.Write(response)
						continue
					}
					_, err = dispatcherConn.Write(ticketBytes)
					if err != nil {
						continue
					}

					for i, t := range s.outgoingTickets {
						if t == ticket {
							s.outgoingTickets = append(s.outgoingTickets[:i], s.outgoingTickets[i+1:]...)
							continue ticketLoop
						}
					}
					continue ticketLoop
				}
			}
		}
	}
}
