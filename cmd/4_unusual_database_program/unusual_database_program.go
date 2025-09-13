package main

import (
	"TDMR87/go_protohackers/internal/server"
	"net"
	"strings"
	"sync"
)

func main() {
	conn, _ := server.StartUdpListener(":8080", handle)
	defer conn.Close()
	select {}
}

func handle(conn *net.UDPConn, buf []byte, n int, clientAddr *net.UDPAddr) {
	msg := string(buf[:n])
	
	if msg == "version" {
		val := db.Retrieve(msg)
		response := msg + "=" + val
		conn.WriteToUDP([]byte(response), clientAddr)
	} else if ContainsEqualsSign(msg) {
		key, val := parse(msg)
		if key != "version" {
			db.Insert(key, val)
		}
	} else {
		val := db.Retrieve(msg)
		response := msg + "=" + val
		conn.WriteToUDP([]byte(response), clientAddr)	
	}
}

var db = Database{
    Store: map[string]string{
        "version": "6.6.6",
    },
}

type Database struct {
	Store map[string]string
	Lock sync.RWMutex
}

func (db *Database) Retrieve(key string) string {
	db.Lock.Lock()
	defer db.Lock.Unlock()
	if db.Store[key] != "" {
		return db.Store[key]
	} else {
		return ""
	}
}

func (db *Database) Insert(key string, val string) {
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

func ContainsEqualsSign(msg string) bool {
	for _, r := range msg {
		if r == '=' {
			return true
		}
	}

	return false
}