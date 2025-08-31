package main

import (
	"TDMR87/protohackers/server"
	"bufio"
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"
)

func main() {
	server.StartTcpListener(":8080", handle)
	select {}
}

func handle(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	conn.Write(NewChatMessage("Welcome to budgetchat! What shall I call you?"))

	username := GetUsername(scanner)
	if !validUsername.MatchString(username) {
		conn.Write(NewChatMessage("Invalid username. Usernames must be 1-16 characters long " +
			"and must consist entirely of alphanumeric characters (uppercase, lowercase, and digits)"))
		conn.Close()
		return
	}

	chatroom.AddUser(username, conn)
	chatroom.SendWelcomeMessage(username)
	chatroom.Announce(username, NewChatMessage(fmt.Sprintf("* %s has entered the room", username)))

	for scanner.Scan() {
		msg := scanner.Text()
		chatroom.Relay(username, NewChatMessage(fmt.Sprintf("[%s] %s", username, msg)))
	}

	chatroom.RemoveUser(username, conn)
	chatroom.Announce(username, NewChatMessage(fmt.Sprintf("* %s has left the room", username)))
}

type ChatMessage []byte
type ChatRoom struct {
	JoinedUsers map[string]net.Conn
	Lock        sync.RWMutex
}

var validUsername = regexp.MustCompile(`^[A-Za-z0-9]{1,16}$`)
var chatroom = ChatRoom{
	JoinedUsers: make(map[string]net.Conn),
}

func GetUsername(scanner *bufio.Scanner) (username string) {
	if scanner.Scan() {
		username = scanner.Text()
	}
	return username
}

func (chatroom *ChatRoom) AddUser(user string, conn net.Conn) {
	chatroom.Lock.Lock()
	defer chatroom.Lock.Unlock()
	chatroom.JoinedUsers[user] = conn
}

func (chatroom *ChatRoom) RemoveUser(user string, conn net.Conn) {
	chatroom.Lock.Lock()
	defer chatroom.Lock.Unlock()
	delete(chatroom.JoinedUsers, user)
	conn.Close()
}

func (chatroom *ChatRoom) Announce(from string, msg ChatMessage) {
	chatroom.Lock.Lock()
	defer chatroom.Lock.Unlock()
	for user := range chatroom.JoinedUsers {
		if user == from {
			continue
		}
		chatroom.JoinedUsers[user].Write(msg)
	}
}

func (chatroom *ChatRoom) Relay(from string, msg ChatMessage) {
	chatroom.Lock.Lock()
	defer chatroom.Lock.Unlock()
	for user := range chatroom.JoinedUsers {
		if user == from {
			continue
		}
		chatroom.JoinedUsers[user].Write(msg)
	}
}

func (chatroom *ChatRoom) SendWelcomeMessage(newUser string) {
	chatroom.Lock.Lock()
	defer chatroom.Lock.Unlock()

	var otherUsersInChatRoom []string
	for user := range chatroom.JoinedUsers {
		if user == newUser {
			continue
		}
		otherUsersInChatRoom = append(otherUsersInChatRoom, user)
	}

	chatroom.JoinedUsers[newUser].Write(NewChatMessage(fmt.Sprintf(
		"* The room contains: %s", strings.Join(otherUsersInChatRoom, ", "))))
}

func NewChatMessage(msg string) ChatMessage {
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	return ChatMessage([]byte(msg))
}
