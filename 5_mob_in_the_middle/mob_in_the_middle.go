package main

import (
	"TDMR87/protohackers/server"
	"bufio"
	"net"
	"sync"
	"time"

	"github.com/dlclark/regexp2"
)

var budgetChatServerAddr = "chat.protohackers.com:16963"
var tonysBogusCoinAddr = "7YWHMfk9JZe0LM0g1ZauHuiSxhI"
var bogusCoinRegex = regexp2.MustCompile(`(?<!\S)7[a-zA-Z0-9]{25,34}(?!\S)`, 0)

func main() {
	bogusCoinRegex.MatchTimeout = time.Second * 5
	conn, _ := server.StartTcpListener(":8080", handle)
	defer conn.Close()
	select {}
}

func handle(
	clientConn net.Conn) {
	upstreamConn, _ := net.Dial("tcp", budgetChatServerAddr)
	defer clientConn.Close()
	defer upstreamConn.Close()

	clientReader := bufio.NewReader(clientConn)
	upstreamReader := bufio.NewReader(upstreamConn)

	var wg sync.WaitGroup

	wg.Go(func() {
		for {
			msg, err := clientReader.ReadString('\n')
			if err != nil { break } // EOF or error, discard partial lines
			msg, _ = bogusCoinRegex.Replace(msg, tonysBogusCoinAddr, -1, -1)
			upstreamConn.Write([]byte(msg))
		}
		upstreamConn.Close()
	})

	wg.Go(func() {
		for {
			msg, err := upstreamReader.ReadString('\n')
			if err != nil { break } // EOF or error, discard partial lines
			msg, _ = bogusCoinRegex.Replace(msg, tonysBogusCoinAddr, -1, -1)
			clientConn.Write([]byte(msg))
		}
		clientConn.Close()
	})

	wg.Wait()
}