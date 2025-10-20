package main

import (
	"TDMR87/go_protohackers/internal/server"
	"net"
	"sync"
	"time"
)

var (
	heartbeatMu      sync.Mutex
	heartbeatClients = make(map[net.Conn]struct{})
	dispatchersMu sync.RWMutex
	dispatchers  = make(map[net.Conn]IAmDispatcher)
	cameraClientsMu      sync.RWMutex
	cameraClients = make(map[net.Conn]IAmCamera)
	snapshotsMu           sync.RWMutex
	cameraPlateSnapshots = make(map[IAmCamera]Plate)
	sentTicketsMu sync.Mutex
	sentTickets   = make(map[string][]uint32)
	outgoingTicketsMu        sync.Mutex
	outgoingTickets = make([]Ticket, 0)
)

func main() {
	server.StartTcpListener(":8080", handle)
	select {}
}

func handle(conn net.Conn) {
	defer conn.Close()
	defer func() {
		heartbeatMu.Lock()
		delete(heartbeatClients, conn)
		heartbeatMu.Unlock()

		cameraClientsMu.Lock()
		delete(cameraClients, conn)
		cameraClientsMu.Unlock()

		dispatchersMu.Lock()
		delete(dispatchers, conn)
		dispatchersMu.Unlock()
	}()

	reader := NewMessageReader(conn)

	for {
		message, err := reader.NextMessage()
		if err != nil {
			response, _ := Error{Msg: err.Error()}.Encode()
			conn.Write(response)
			return
		}

		switch msg := message.(type) {
		case WantHeartBeat:
			heartbeatMu.Lock()
			_, exists := heartbeatClients[conn]
			heartbeatMu.Unlock()

			if exists {
				response, _ := Error{Msg: "Client is already receiving heartbeats"}.Encode()
				conn.Write(response)
				return
			}
			if msg.Interval > 0 {
				go sendHeartBeat(conn, msg.Interval)
			}
			continue
		case IAmCamera:
			cameraClientsMu.Lock()
			_, exists := cameraClients[conn]
			if exists {
				cameraClientsMu.Unlock()
				response, _ := Error{Msg: "Client is already identified as a camera"}.Encode()
				conn.Write(response)
				return
			}
			cameraClients[conn] = msg
			cameraClientsMu.Unlock()
			continue
		case Plate:
			cameraClientsMu.RLock()
			camera, exists := cameraClients[conn]
			cameraClientsMu.RUnlock()

			if exists {
				snapshotsMu.Lock()
				cameraPlateSnapshots[camera] = msg
				snapshotsMu.Unlock()
				go handlePlate(msg, camera)
				continue
			}
			response, _ := Error{Msg: "Client must be identified as a camera to send a plate"}.Encode()
			conn.Write(response)
			continue
		case IAmDispatcher:
			dispatchersMu.Lock()
			_, exists := dispatchers[conn]
			if exists {
				dispatchersMu.Unlock()
				response, _ := Error{Msg: "Client is already identified as a dispatcher"}.Encode()
				conn.Write(response)
				return
			}
			dispatchers[conn] = msg
			dispatchersMu.Unlock()
			go sendTickets()
			continue
		default:
			response, _ := Error{Msg: "Unknown message received from MessageReader"}.Encode()
			conn.Write(response)
			continue
		}
	}
}

func sendHeartBeat(conn net.Conn, deciSeconds uint32) {
	heartbeatMu.Lock()
	heartbeatClients[conn] = struct{}{}
	heartbeatMu.Unlock()

	interval := time.Duration(deciSeconds*100) * time.Millisecond
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	defer func() {
		heartbeatMu.Lock()
		delete(heartbeatClients, conn)
		heartbeatMu.Unlock()
	}()

	for range ticker.C {
		_, err := conn.Write(HeartBeat{}.Encode())
		if err != nil {
			return
		}
	}
}

func handlePlate(currentPlate Plate, currentCamera IAmCamera) {
outerloop:
	for previousCamera, previousCameraPlate := range cameraPlateSnapshots {
		if previousCamera == currentCamera ||
			previousCameraPlate.Plate != currentPlate.Plate ||
			previousCamera.Road != currentCamera.Road {
			continue
		}

		currentDay := currentPlate.Timestamp / 86400
		previousDay := previousCameraPlate.Timestamp / 86400

		if _, exists := sentTickets[currentPlate.Plate]; exists {
			for _, day := range sentTickets[currentPlate.Plate] {
				if day >= previousDay && day <= currentDay {
					continue outerloop // Ticket already sent for this day and plate
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
			sentTicketsMu.Lock()
			defer sentTicketsMu.Unlock()
			sentTickets[currentPlate.Plate] = append(sentTickets[currentPlate.Plate], day)
		}

		if currentCamera.Mile > previousCamera.Mile {
			outgoingTickets = append(outgoingTickets, Ticket{
				Plate:      currentPlate.Plate,
				Road:       currentCamera.Road,
				Mile1:      previousCamera.Mile,
				Timestamp1: previousCameraPlate.Timestamp,
				Mile2:      currentCamera.Mile,
				Timestamp2: currentPlate.Timestamp,
				Speed:      uint16(speedInMph) * 100,
			})
		} else {
			outgoingTickets = append(outgoingTickets, Ticket{
				Plate:      currentPlate.Plate,
				Road:       currentCamera.Road,
				Mile1:      currentCamera.Mile,
				Timestamp1: currentPlate.Timestamp,
				Mile2:      previousCamera.Mile,
				Timestamp2: previousCameraPlate.Timestamp,
				Speed:      uint16(speedInMph) * 100,
			})
		}

		go sendTickets()
		break
	}
}

func sendTickets() {
	ticketsCopy := make([]Ticket, len(outgoingTickets))
	copy(ticketsCopy, outgoingTickets)

ticketLoop:
	for _, ticket := range ticketsCopy {
		for dispatcherConn, dispatcher := range dispatchers {
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

					// Remove ticket from original slice
					for i, t := range outgoingTickets {
						if t == ticket {
							outgoingTickets = append(outgoingTickets[:i], outgoingTickets[i+1:]...)
							continue ticketLoop
						}
					}
					continue ticketLoop
				}
			}
		}
	}
}
