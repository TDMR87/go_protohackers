package main

import (
	"TDMR87/protohackers/server"
	"bufio"
	"encoding/json"
	"log"
	"math"
	"net"
)

type Request struct {
	Method string  `json:"method"`
	Number *float64 `json:"number"`
}

type Response struct {
	Method string `json:"method"`
	Prime  bool   `json:"prime"`
}

func main() {
	server.StartTcpListener(":8080", handle)
	select {}
}

func handle(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	
	for scanner.Scan() {
		var req Request
		err := json.Unmarshal([]byte(scanner.Text()), &req)

		if err != nil || req.Method != "isPrime" || req.Number == nil {
			log.Println("Malformed request:", scanner.Text())
			conn.Write([]byte("malformed\n"))
			conn.Close()
			return
		}

		response, _ := json.Marshal(Response{
			Method: "isPrime",
			Prime:  isPrime(*req.Number)})

		conn.Write(append(response, '\n'))
	}
}

func isPrime(num float64) bool {
	// Only positive integers can be prime
	if num != math.Trunc(num) || num < 2 {
		return false
	}

	// 2 is always prime
	n := int(num)
	if n == 2 {
		return true
	}

	// If n is divisible by 2, not a prime
	if n%2 == 0 {
		return false
	}

	// Start from 3 up until the square root of n.
	// If we haven’t found a divisor by the time we've checked up to square root of n, 
	// there can’t be a prime beyond that.
	// Also, increment by 2 to skip even numbers.
	for i := 3; i*i <= n; i += 2 {
		if n%i == 0 {
			// Found a divisor, not prime
			return false 
		}
	}

	// No divisors found, is prime
	return true 
}
