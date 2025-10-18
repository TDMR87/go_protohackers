package main

import (
	"TDMR87/go_protohackers/internal/server"
	"fmt"
	"net"
	"time"
)

var heartbeatClients = make(map[net.Conn]struct{})
var cameraClients = make(map[net.Conn]IAmCamera)
var cameraPlateSnapshots = make(map[IAmCamera]Plate)
var outgoingTickets = make([]Ticket, 0)

func main() {
	server.StartTcpListener(":8080", handle)
}

func handle(conn net.Conn) {
	defer conn.Close()
	defer delete(heartbeatClients, conn)
	defer delete(cameraClients, conn)

	reader := NewMessageReader(conn)

	for {
		message, err := reader.NextMessage()
		if err != nil {
			response, _ := Error{Msg: err.Error()}.Encode()
			conn.Write(response)
			conn.Close()
			return
		}

		switch msg := message.(type) {
		case WantHeartBeat:
			if _, exists := heartbeatClients[conn]; exists {
				response, _ := Error{Msg: "Client is already receiving heartbeats"}.Encode()
				conn.Write(response)
				conn.Close()
				return
			}
			if msg.Interval > 0 {
				go sendHeartBeat(conn, msg.Interval)
				continue
			}
		case IAmCamera:
			if _, exists := cameraClients[conn]; exists {
				response, _ := Error{Msg: "Client is already identified as a camera"}.Encode()
				conn.Write(response)
				conn.Close()
				return
			}
			cameraClients[conn] = msg
			continue
		case Plate:
			if camera, exists := cameraClients[conn]; exists {
				cameraPlateSnapshots[camera] = msg
				go handlePlate(msg, camera)
				continue
			}
			response, _ := Error{Msg: "Client must be identified as a camera to send a plate"}.Encode()
			conn.Write(response)
			conn.Close()
			return
		case IAmDispatcher:
			response, _ := Error{Msg: "Not implemented"}.Encode()
			conn.Write(response)
			conn.Close()
			return
		default:
			fmt.Printf("Unknown message type: %T\n", msg)
		}
	}
}

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

func handlePlate(currentPlate Plate, currentCamera IAmCamera) {
	for previousCamera, previousCameraPlate := range cameraPlateSnapshots {
		if previousCamera.Road == currentCamera.Road &&
			previousCamera.Mile < currentCamera.Mile &&
			previousCameraPlate.Plate == currentPlate.Plate {

			distanceDiff := currentCamera.Mile - previousCamera.Mile
			timeDiff := currentPlate.Timestamp - previousCameraPlate.Timestamp
			if timeDiff == 0 {
				continue
			}

			speedInMph := (float64(distanceDiff) / float64(timeDiff)) * 3600.0
			if speedInMph < float64(currentCamera.Limit) {
				continue
			}
			
			outgoingTickets = append(outgoingTickets, Ticket{
				Plate:      currentPlate.Plate,
				Road:       currentCamera.Road,
				Mile1:      previousCamera.Mile,
				Timestamp1: previousCameraPlate.Timestamp,
				Mile2:      currentCamera.Mile,
				Timestamp2: currentPlate.Timestamp,
				Speed:      uint16(speedInMph) * 100,
			})

			go sendTickets()

			break // Only consider the first matching camera
		}
	}
}

func sendTickets() {
	// for _, ticket := range outgoingTickets {

	// }
}