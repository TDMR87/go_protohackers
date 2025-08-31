package main

import (
	"TDMR87/protohackers/server"
	"log"
	"net"
	"strings"
	"sync"
)

func main() {
	conn, _ := server.StartUdpListener(":8080", handle)
	defer conn.Close()
	select {}
}

func handle(conn *net.UDPConn, buf []byte, n int, clientAddr *net.UDPAddr, err error) {
	if err != nil {
		log.Println(err.Error())
		return
	}

	msg := string(buf[:n])
	
	if isInsert(msg) {
		key, val := parse(msg)
		db.Set(key, val)
	} else if msg == "version" {
		response  := msg + "=" + "Ken's Key-Value Store 1.0"
		conn.WriteToUDP([]byte(response), clientAddr)	
	} else {
		val := db.Get(msg)
		response := msg + "=" + val
		conn.WriteToUDP([]byte(response), clientAddr)	
	}
}

var db = Database{
	Store: make(map[string]string),
}

type Database struct {
	Store map[string]string
	Lock sync.RWMutex
}

func (db *Database) Get(key string) string {
	db.Lock.Lock()
	defer db.Lock.Unlock()
	return db.Store[key]
}

func (db *Database) Set(key string, val string) {
	db.Lock.Lock()
	defer db.Lock.Unlock()
	db.Store[key] = val
}

func parse(msg string) (key string, val string) {
	if k, v, ok := strings.Cut(msg, "="); ok {
		key = k
		val = v
	}
	return
}

func isInsert(msg string) bool {
	for _, r := range msg {
		if r == '=' {
			return true
		}
	}

	return false
}