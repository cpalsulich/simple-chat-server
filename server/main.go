package main

import (
	"bufio"
	"encoding/gob"
	chat "github.com/cpalsulich/simple-chat-server"
	"log"
	"net"
)

func main() {
	clients := make(map[string]*chat.User)
	rooms := make(map[string]*chat.Room)

	if err := startServer(clients, rooms); err != nil {
		panic(err)
	}
}

type server struct {
	rooms         map[string]*chat.Room
	users         map[string]*chat.User
	creating      chan chat.Room
	connecting    chan chat.User
	disconnecting chan chat.User
}

func startServer(users map[string]*chat.User, rooms map[string]*chat.Room) error {
	server := server{
		rooms:         rooms,
		users:         users,
		creating:      make(chan chat.Room, 10),
		connecting:    make(chan chat.User, 10),
		disconnecting: make(chan chat.User, 10),
	}
	go server.consumeCreating()
	go server.consumeConnecting()
	go server.consumeDisconnecting()

	endpoint := chat.NewEndpoint()
	endpoint.AddHandleFunc(chat.GetRooms, server.handleGetRooms)
	endpoint.AddHandleFunc(chat.JoinRoom, server.handleJoinRoom)
	endpoint.AddHandleFunc(chat.Post, server.handlePost)
	endpoint.AddHandleFunc(chat.LeaveRoom, server.handleLeaveRoom)
	endpoint.AddHandleFunc(chat.CreateRoom, server.handleCreateRoom)
	return endpoint.Listen("5001", server.handleConnect, server.handleDisconnect)
}

func (s *server) consumeCreating() {
	for {
		r, ok := <-s.creating
		if ok == false {
			return
		}
		log.Println("new room")
		log.Println("room name " + r.Name)
		s.rooms[r.Name] = chat.NewRoom(r.Name)
	}
}

func (s *server) consumeConnecting() {
	for {
		u, ok := <-s.connecting
		if ok == false {
			return
		}
		log.Printf("user connected %v", u)
		s.users[u.ID] = &u
	}
}

func (s *server) consumeDisconnecting() {
	for {
		u, ok := <-s.disconnecting
		if ok == false {
			return
		}
		log.Printf("user disconnected %v", u)
		delete(s.users, u.ID)
		log.Printf("num users: %d", len(s.users))
	}
}

func (s *server) handleConnect(conn *net.Conn) {
	user := chat.NewUser(conn)
	s.connecting <- *user
}

func (s *server) handleDisconnect(conn *net.Conn) {
	user := chat.NewUser(conn)
	s.disconnecting <- *user
}

func (s *server) handleGetRooms(rw *bufio.ReadWriter, conn *net.Conn) {
	keys := make([]string, len(s.rooms))
	i := 0
	for k := range s.rooms {
		keys[i] = k
		i++
	}

	log.Println("sending get rooms response")
	enc := gob.NewEncoder(rw)

	if err := enc.Encode(chat.Action{Name: chat.GetRooms}); err != nil {
		chat.LogError("failed to encode action: %w", err)
	}

	if err := enc.Encode(keys); err != nil {
		chat.LogError("failed to encode keys: %w", err)
	}

	if err := rw.Flush(); err != nil {
		chat.LogError("failed to flush: %w", err)
	}
}

func (s *server) handleJoinRoom(rw *bufio.ReadWriter, conn *net.Conn) {
	r := &chat.Room{}

	if err := gob.NewDecoder(rw).Decode(r); err != nil {
		chat.LogError("failed to decode room: %w", err)
	}

	room := s.rooms[r.Name]
	if room == nil {
		log.Println("joining non-existent room " + r.Name)
		return
	}
	u := chat.NewUser(conn)
	log.Printf("user %s joining room %s", u, r)
	room.Join(s.users[u.ID])
	enc := gob.NewEncoder(rw)

	if err := enc.Encode(chat.Action{Name: chat.JoinRoom}); err != nil {
		chat.LogError("failed to encode action: %w", err)
	}

	if err := enc.Encode(r); err != nil {
		chat.LogError("failed to encode room: %w", err)
	}

	if err := rw.Flush(); err != nil {
		chat.LogError("failed to flush: %w", err)
	}
}

func (s *server) handlePost(rw *bufio.ReadWriter, conn *net.Conn) {
	msg := &chat.Message{}

	if err := gob.NewDecoder(rw).Decode(msg); err != nil {
		chat.LogError("failed to decode message: %w", err)
	}

	r := s.rooms[msg.Room]
	if r == nil {
		log.Printf("couldn't find room (%s) for post\n", msg.Room)
		return
	}
	msg.Author = chat.NewUser(conn).ID
	r.Post(msg)
	log.Printf("post message %s to room %s", msg.Message, msg.Room)
}

func (s *server) handleLeaveRoom(rw *bufio.ReadWriter, conn *net.Conn) {
	r := &chat.Room{}
	if err := gob.NewDecoder(rw).Decode(r); err != nil {
		chat.LogError("failed to decode room: %w", err)
		return
	}

	room := s.rooms[r.Name]
	c := chat.NewUser(conn)
	room.Leave(s.users[c.ID])
	enc := gob.NewEncoder(rw)

	if err := enc.Encode(chat.Action{Name: chat.LeaveRoom}); err != nil {
		chat.LogError("failed to encode action: %w", err)
		return
	}

	if err := enc.Encode(room); err != nil {
		chat.LogError("failed to encode room: %w", err)
		return
	}

	if err := rw.Flush(); err != nil {
		chat.LogError("failed to flush: %w", err)
	}
}

func (s *server) handleCreateRoom(rw *bufio.ReadWriter, conn *net.Conn) {
	r := &chat.Room{}
	if err := gob.NewDecoder(rw).Decode(r); err != nil {
		chat.LogError("failed to decode room: %w", err)
		return
	}
	s.creating <- *r
}
