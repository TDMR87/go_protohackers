package main

import (
	"TDMR87/protohackers/server"
	"bufio"
	"encoding/binary"
	"net"

	"github.com/google/uuid"
)

func main() {
	server.StartTcpListener(":8080", handle)
	select {}
}

var sessionData = SessionData{}

func handle(conn net.Conn) {
	defer conn.Close()
	sessionId := SessionId(uuid.New())
	scanner := bufio.NewScanner(conn)
	scanner.Split(messageSplitter)

	for scanner.Scan() {
		bytes := []byte(scanner.Bytes())
		switch bytes[0] {
		case 'I':
			handleInsert(InsertMessage(bytes), sessionId)
		case 'Q':
			queryResult := handleQuery(QueryMessage(bytes), sessionId)
			conn.Write(queryResult)
		default:
			conn.Write([]byte("malformed"))
			conn.Close()
			return
		}
	}
}

type QueryMessage []byte
type InsertMessage []byte
type SessionId uuid.UUID
type SessionData map[SessionId][]Price
type Price struct {
	Timestamp int32
	Price     int32
}

func handleInsert(msg InsertMessage, sessionId SessionId) {
	sessionData[sessionId] = append(sessionData[sessionId], Price{
		msg.timestamp(),
		msg.price(),
	})
}

func handleQuery(msg QueryMessage, sessionId SessionId) (queryResult []byte) {
	var prices []Price
	for _, price := range sessionData[sessionId] {
		if price.Timestamp >= msg.minTime() && price.Timestamp <= msg.maxTime() {
			prices = append(prices, price)
		}
	}

	if len(prices) == 0 {
		queryResult = make([]byte, 4)
		binary.BigEndian.PutUint32(queryResult, 0)
		return
	}

	var pricesSum int64 // The sum may exceed int32
	for _, price := range prices {
		pricesSum += int64(price.Price)
	}

	mean := pricesSum / int64(len(prices))
	queryResult = make([]byte, 4)
	binary.BigEndian.PutUint32(queryResult, uint32(mean))
	return
}

func messageSplitter(data []byte, atEOF bool) (advance int, token []byte, err error) {
	const chunkSize = 9
	if len(data) < chunkSize {
		return 0, nil, nil
	}

	token = data[:chunkSize]
	advance = chunkSize
	err = nil
	return
}

func (q *QueryMessage) minTime() int32 {
	return int32(binary.BigEndian.Uint32((*q)[1:5]))
}

func (q *QueryMessage) maxTime() int32 {
	return int32(binary.BigEndian.Uint32((*q)[5:9]))
}

func (q *InsertMessage) timestamp() int32 {
	return int32(binary.BigEndian.Uint32((*q)[1:5]))
}

func (q *InsertMessage) price() int32 {
	return int32(binary.BigEndian.Uint32((*q)[5:9]))
}
